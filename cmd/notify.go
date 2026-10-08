package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/USA-RedDragon/configulator/v2"
	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/spf13/cobra"
)

func newNotifyCommand(version, commit string) *cobra.Command {
	return &cobra.Command{
		Use:     "notify",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Short:   "Notify the daemon of a change in the mesh",
		Annotations: map[string]string{
			annotationVersion: version,
			annotationCommit:  commit,
		},
		RunE:              runNotify,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}
}

func runNotify(cmd *cobra.Command, _ []string) error {
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

	if !config.OLSR {
		return nil
	}

	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPost, fmt.Sprintf("http://0.0.0.0:%d/notify", config.Port), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error notifying daemon: %s", resp.Status)
	}

	return nil
}
