package services

import (
	"log/slog"

	"github.com/puzpuzpuz/xsync/v4"
	"golang.org/x/sync/errgroup"
)

type Registry struct {
	services *xsync.Map[string, Service]
}

type ServiceName string

const (
	OLSRServiceName     ServiceName = "olsr"
	BabelServiceName    ServiceName = "babel"
	DNSMasqServiceName  ServiceName = "dnsmasq"
	MeshLinkServiceName ServiceName = "meshlink"
	LQMServiceName      ServiceName = "lqm"
)

func NewServiceRegistry() *Registry {
	return &Registry{
		services: xsync.NewMap[string, Service](),
	}
}

func (r *Registry) Register(name ServiceName, service Service) {
	r.services.Store(string(name), service)
}

func (r *Registry) Get(name ServiceName) (Service, bool) {
	return r.services.Load(string(name))
}

func (r *Registry) StartAll() {
	r.services.Range(func(name string, service Service) bool {
		if !service.IsEnabled() {
			slog.Debug("service is disabled", "service", name)
			return true
		}
		go func() {
			for {
				err := service.Start()
				if err != nil {
					slog.Warn("service failed to start", "service", name, "error", err)
				}
			}
		}()
		return true
	})
}

func (r *Registry) StopAll() error {
	errGrp := errgroup.Group{}
	r.services.Range(func(_ string, service Service) bool {
		errGrp.Go(func() error {
			return service.Stop()
		})
		return true
	})
	return errGrp.Wait()
}
