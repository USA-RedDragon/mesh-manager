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
	respChan, err := walk.Walk(config.ServerName)
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

	w := bufio.NewWriter(responsesFile)

	n, err := w.Write([]byte("["))
	if err != nil {
		return fmt.Errorf("failed to write opening bracket: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("failed to write opening bracket: %w", err)
	}
	enc := json.NewEncoder(w)

	type nodeEntry struct {
		Data apimodels.SysinfoResponse `json:"data"`
	}

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
					for key, value := range linkInfo {
						if value.LinkType == apimodels.LinkTypeTun || value.LinkType == apimodels.LinkTypeWireguard {
							value.LinkType = apimodels.LinkTypeSupernode
							linkInfo[key] = value
						}
					}
					resp.SetLinkInfo(linkInfo)
				}
			}

			err = enc.Encode(nodeEntry{
				Data: *resp,
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
	output := map[string]interface{}{
		"nonMapped":    nonMapped,
		"hostsScraped": walk.TotalCount.Value(),
		"date":         time.Now().UTC().Format(time.RFC3339),
	}

	slog.Info("output", "output", output)

	return nil
}
