package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/USA-RedDragon/configulator"
	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
	"github.com/USA-RedDragon/mesh-manager/internal/walker/walker"
	"github.com/spf13/cobra"
)

func newWalkCommand(version, commit string) *cobra.Command {
	return &cobra.Command{
		Use:     "walk",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Short:   "Walk the mesh and update the meshmap json",
		Annotations: map[string]string{
			"version": version,
			"commit":  commit,
		},
		RunE:              runWalk,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}
}

//nolint:gocyclo
func runWalk(cmd *cobra.Command, _ []string) error {
	err := runRoot(cmd, nil)
	if err != nil {
		slog.Error("Encountered an error.", "error", err.Error())
	}

	ctx := cmd.Context()

	c, err := configulator.FromContext[config.Config](ctx)
	if err != nil {
		return fmt.Errorf("failed to get config from context")
	}

	config, err := c.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !config.Walker {
		return fmt.Errorf("walker is not enabled in the configuration")
	}

	walk := walker.NewWalker(2*time.Minute, 5, 5*time.Second)
	respChan, err := walk.Walk(ctx, config.ServerName)
	if err != nil {
		return fmt.Errorf("failed to start walk: %w", err)
	}

	slog.Info("Starting walk", "startingNode", config.ServerName)

	nonMapped := 0
	completed := 0
	go func() {
		for range time.Tick(2 * time.Second) {
			total := walk.TotalCount.Value()
			mapped := completed - nonMapped
			slog.Info("Still walking", "completed", completed, "total", total, "mapped", mapped, "unmapped", nonMapped)
		}
	}()

	responsesFile, err := os.CreateTemp(os.TempDir(), "responses.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = responsesFile.Close()
		_ = os.Remove(responsesFile.Name())
	}()

	w := bufio.NewWriter(responsesFile)

	n, err := w.Write([]byte("["))
	if err != nil {
		return fmt.Errorf("failed to write opening bracket: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("failed to write opening bracket: %w", err)
	}
	enc := json.NewEncoder(w)

	for resp := range respChan {
		completed++
		if resp == nil {
			continue
		}
		lat := resp.GetLatitude()
		lon := resp.GetLongitude()
		if lat != 0 && lon != 0 {
			if resp.GetMeshSupernode() {
				linkInfo := resp.GetLinkInfo()
				if linkInfo != nil {
					switch v := linkInfo.(type) {
					case map[string]apimodels.LinkInfo1Point7:
						for key, value := range v {
							if value.LinkType == apimodels.LinkTypeTun || value.LinkType == apimodels.LinkTypeWireguard {
								value.LinkType = apimodels.LinkTypeSupernode
								v[key] = value
							}
						}
						resp.SetLinkInfo(v)
					case map[string]apimodels.LinkInfo2Point0:
						for key, value := range v {
							if value.LinkType == apimodels.LinkTypeTun || value.LinkType == apimodels.LinkTypeWireguard {
								value.LinkType = apimodels.LinkTypeSupernode
								v[key] = value
							}
						}
						resp.SetLinkInfo(v)
					}
				}
			}

			err = enc.Encode(map[string]any{
				"data": resp.GetObject(),
			})
			if err != nil {
				return fmt.Errorf("failed to encode response: %w", err)
			}
			n, err = w.Write([]byte(","))
			if err != nil {
				return fmt.Errorf("failed to write comma: %w", err)
			}
			if n != 1 {
				return fmt.Errorf("failed to write comma: %w", err)
			}
		} else {
			nonMapped++
		}
	}

	err = w.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush responses: %w", err)
	}

	// We now need to delete the last comma and replace it with a closing bracket
	_, err = responsesFile.Seek(-1, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}
	written, err := responsesFile.Write([]byte("]"))
	if err != nil {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}
	if written != 1 {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}
	err = responsesFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync output file: %w", err)
	}

	slog.Info("Finished walking")

	// Save output
	output := map[string]any{
		"nonMapped":    nonMapped,
		"hostsScraped": walk.TotalCount.Value(),
		"date":         time.Now().UTC().Format(time.RFC3339),
	}

	err = createFile(output, responsesFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	return nil
}

func createFile(output map[string]any, responsesFile *os.File) error {
	file, err := os.Create("/meshmap/data/out.json.new")
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}

	if err := json.NewEncoder(file).Encode(output); err != nil {
		return fmt.Errorf("error encoding output file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("error closing output file: %w", err)
	}

	// Now we need to combine out.json.new and responses.json
	file, err = os.OpenFile("/meshmap/data/out.json.new", os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error opening output file: %w", err)
	}

	// Seek to before the closing bracket
	_, err = file.Seek(-2, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("error seeking to before closing bracket: %w", err)
	}
	// Replace the closing bracket
	n, err := file.Write([]byte(",\"nodeInfo\":"))
	if err != nil {
		return fmt.Errorf("error writing nodeInfo key: %w", err)
	}
	if n != 12 {
		return fmt.Errorf("error writing nodeInfo key: %w", err)
	}
	_, err = responsesFile.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("error seeking to start of responses file: %w", err)
	}
	r := bufio.NewReader(responsesFile)
	_, err = io.Copy(file, r)
	if err != nil {
		return fmt.Errorf("error copying responses file to output file: %w", err)
	}
	// Write the closing bracket
	n, err = file.Write([]byte("}"))
	if err != nil {
		return fmt.Errorf("error writing closing bracket: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("error writing closing bracket: %w", err)
	}
	err = file.Sync()
	if err != nil {
		return fmt.Errorf("error syncing output file: %w", err)
	}
	err = responsesFile.Close()
	if err != nil {
		return fmt.Errorf("error closing responses file: %w", err)
	}

	err = os.Rename("/meshmap/data/out.json.new", "/meshmap/data/out.json")
	if err != nil {
		return fmt.Errorf("error renaming output file: %w", err)
	}

	return nil
}
