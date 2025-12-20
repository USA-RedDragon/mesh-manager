package olsr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/USA-RedDragon/mesh-manager/internal/config"
)

type Service struct {
	config  *config.Config
	olsrCmd *exec.Cmd
}

func NewService(config *config.Config) *Service {
	return &Service{
		config:  config,
		olsrCmd: exec.CommandContext(context.Background(), "olsrd", "-f", "/etc/olsrd/olsrd.conf", "-nofork"),
	}
}

func (s *Service) Start() error {
	if s.olsrCmd.Process != nil && s.olsrCmd.ProcessState == nil {
		return s.olsrCmd.Wait()
	}
	if s.olsrCmd.ProcessState != nil {
		s.olsrCmd = exec.CommandContext(context.Background(), "olsrd", "-f", "/etc/olsrd/olsrd.conf", "-nofork")
	}
	err := s.olsrCmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}
	return s.olsrCmd.Wait()
}

func (s *Service) Stop() error {
	if s.olsrCmd != nil && s.olsrCmd.Process != nil {
		err := s.olsrCmd.Process.Signal(os.Signal(syscall.SIGTERM))
		if err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}
	return nil
}

func (s *Service) Reload() error {
	return s.olsrCmd.Process.Signal(os.Signal(syscall.SIGHUP))
}

func (s *Service) IsRunning() bool {
	return s.olsrCmd != nil && s.olsrCmd.Process != nil
}

func (s *Service) IsEnabled() bool {
	return true
}
