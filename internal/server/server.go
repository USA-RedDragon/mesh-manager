package server

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/USA-RedDragon/mesh-manager/internal/bandwidth"
	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/USA-RedDragon/mesh-manager/internal/events"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/middleware"
	"github.com/USA-RedDragon/mesh-manager/internal/services"
	"github.com/USA-RedDragon/mesh-manager/internal/services/meshlink"
	"github.com/USA-RedDragon/mesh-manager/internal/services/olsr"
	"github.com/USA-RedDragon/mesh-manager/internal/wireguard"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/sessions"
	gormsessions "github.com/gin-contrib/sessions/gorm"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/pbkdf2"
	"gorm.io/gorm"
)

const defTimeout = 10 * time.Second
const debugWriteTimeout = 60 * time.Second

type Server struct {
	config           *config.Config
	server           *http.Server
	db               *gorm.DB
	shutdownChannel  chan bool
	stats            *bandwidth.StatCounterManager
	eventsChannel    chan events.Event
	wireguardManager *wireguard.Manager
}

func NewServer(config *config.Config, db *gorm.DB, stats *bandwidth.StatCounterManager, eventsChannel chan events.Event, wireguardManager *wireguard.Manager) *Server {
	return &Server{
		config:           config,
		db:               db,
		shutdownChannel:  make(chan bool),
		stats:            stats,
		eventsChannel:    eventsChannel,
		wireguardManager: wireguardManager,
	}
}

func (s *Server) Run(version string, registry *services.Registry) error {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	s.addMiddleware(r, version, registry)

	api.ApplyRoutes(r, s.eventsChannel, s.config)

	err := r.SetTrustedProxies(s.config.TrustedProxies)
	if err != nil {
		slog.Error("Failed to set trusted proxies", "error", err)
	}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", "127.0.0.1", s.config.Port),
		Handler:      r,
		ReadTimeout:  defTimeout,
		WriteTimeout: 0,
	}
	server.SetKeepAlivesEnabled(false)

	s.server = server

	go s.run()

	return nil
}

func (s *Server) run() {
	func() {
		err := s.server.ListenAndServe()
		if err != nil {
			switch {
			case errors.Is(err, http.ErrServerClosed):
				s.shutdownChannel <- true
				return
			default:
				slog.Error("Failed to start HTTP server", "error", err)
			}
		}
	}()
}

func (s *Server) Stop() error {
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}
	<-s.shutdownChannel
	return nil
}

func (s *Server) addMiddleware(r *gin.Engine, version string, registry *services.Registry) {
	// Debug
	if s.config.PProf.Enabled {
		pprof.Register(r)
	}

	var di = &middleware.DepInjection{
		Config:           s.config,
		DB:               s.db,
		NetworkStats:     s.stats,
		ServiceRegistry:  registry,
		Version:          version,
		WireguardManager: s.wireguardManager,
	}

	if s.config.Babel.Enabled {
		di.MeshLinkParser = meshlink.NewParser()
	}
	if s.config.OLSR {
		di.OLSRHostsParser = olsr.NewHostsParser()
		di.OLSRServicesParser = olsr.NewServicesParser()
	}

	r.Use(middleware.Inject(di))

	// CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowCredentials = true
	if len(s.config.CORSHosts) == 0 {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowAllOrigins = false
		corsConfig.AllowOrigins = s.config.CORSHosts
	}
	r.Use(cors.New(corsConfig))

	// Sessions
	const iterations = 4096
	const keyLen = 32

	sessionStore := gormsessions.NewStore(s.db, true, pbkdf2.Key(
		[]byte(s.config.SessionSecret),
		[]byte(s.config.PasswordSalt),
		iterations,
		keyLen,
		sha256.New,
	))
	r.Use(sessions.Sessions("sessions", sessionStore))
}
