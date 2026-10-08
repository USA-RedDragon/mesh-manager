package cmd

import (
	"fmt"
	"log/slog"

	"github.com/USA-RedDragon/configulator/v2"
	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/USA-RedDragon/mesh-manager/internal/db"
	"github.com/USA-RedDragon/mesh-manager/internal/services/babel"
	"github.com/USA-RedDragon/mesh-manager/internal/services/olsr"
	"github.com/spf13/cobra"
)

func newGenerateCommand(version, commit string) *cobra.Command {
	return &cobra.Command{
		Use:     "generate",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Short:   "Generate olsrd, babeld configs",
		Annotations: map[string]string{
			annotationVersion: version,
			annotationCommit:  commit,
		},
		RunE:              runGenerate,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}
}

func runGenerate(cmd *cobra.Command, _ []string) error {
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

	db, err := db.MakeDB(config)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if config.OLSR {
		slog.Info("Generating olsrd config")
		err = olsr.GenerateAndSave(config, db)
		if err != nil {
			return err
		}
	}

	if config.Babel.Enabled {
		slog.Info("Generating babeld config")
		err = babel.GenerateAndSave(config, db)
		if err != nil {
			return err
		}
	}

	return nil
}
