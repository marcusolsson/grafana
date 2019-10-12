package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/facebookgo/inject"
	"golang.org/x/sync/errgroup"

	"github.com/grafana/grafana/pkg/api"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/bus"
	_ "github.com/grafana/grafana/pkg/extensions"
	"github.com/grafana/grafana/pkg/infra/localcache"
	"github.com/grafana/grafana/pkg/infra/log"
	_ "github.com/grafana/grafana/pkg/infra/metrics"
	_ "github.com/grafana/grafana/pkg/infra/remotecache"
	_ "github.com/grafana/grafana/pkg/infra/serverlock"
	_ "github.com/grafana/grafana/pkg/infra/tracing"
	_ "github.com/grafana/grafana/pkg/infra/usagestats"
	"github.com/grafana/grafana/pkg/login"
	"github.com/grafana/grafana/pkg/login/social"
	"github.com/grafana/grafana/pkg/middleware"
	_ "github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/registry"
	_ "github.com/grafana/grafana/pkg/services/alerting"
	_ "github.com/grafana/grafana/pkg/services/auth"
	_ "github.com/grafana/grafana/pkg/services/cleanup"
	_ "github.com/grafana/grafana/pkg/services/notifications"
	_ "github.com/grafana/grafana/pkg/services/provisioning"
	_ "github.com/grafana/grafana/pkg/services/rendering"
	_ "github.com/grafana/grafana/pkg/services/search"
	_ "github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
)

type Server struct {
	context            context.Context
	shutdownFn         context.CancelFunc
	childRoutines      *errgroup.Group
	log                log.Logger
	cfg                *setting.Cfg
	shutdownReason     string
	shutdownInProgress bool

	configFile string
	homePath   string
	pidFile    string

	RouteRegister routing.RouteRegister `inject:""`
	HttpServer    *api.HTTPServer       `inject:""`
}

func NewServer(configFile, homePath, pidFile string) *Server {
	rootCtx, shutdownFn := context.WithCancel(context.Background())
	childRoutines, childCtx := errgroup.WithContext(rootCtx)

	return &Server{
		context:       childCtx,
		shutdownFn:    shutdownFn,
		childRoutines: childRoutines,
		log:           log.New("server"),
		cfg:           setting.NewCfg(),

		configFile: configFile,
		homePath:   homePath,
		pidFile:    pidFile,
	}
}

func (s *Server) Run() error {
	s.loadConfiguration()
	s.writePIDFile()

	login.Init()
	social.NewOAuthService()

	services := registry.GetServices()

	if err := s.buildServiceGraph(services); err != nil {
		return err
	}

	// Initialize and start services.
	for _, service := range services {
		if registry.IsDisabled(service.Instance) {
			continue
		}

		s.log.Info("Initializing " + service.Name)

		if err := service.Instance.Init(); err != nil {
			return fmt.Errorf("Service init failed: %v", err)
		}
	}

	// Start background services.
	for _, svc := range services {
		// Variable needed for accessing loop variable in function callback.
		descriptor := svc

		service, ok := svc.Instance.(registry.BackgroundService)
		if !ok {
			continue
		}

		if registry.IsDisabled(descriptor.Instance) {
			continue
		}

		s.childRoutines.Go(func() error {
			// Skip starting new service when shutting down
			// Can happen when service stop/return during startup
			if s.shutdownInProgress {
				return nil
			}

			err := service.Run(s.context)

			// If error is not canceled then the service crashed
			if err != context.Canceled && err != nil {
				s.log.Error("Stopped "+descriptor.Name, "reason", err)
			} else {
				s.log.Info("Stopped "+descriptor.Name, "reason", err)
			}

			// Mark that we are in shutdown mode
			// So more services are not started
			s.shutdownInProgress = true
			return err
		})
	}

	sendSystemdNotification("READY=1")

	return s.childRoutines.Wait()
}

func (s *Server) buildServiceGraph(services []*registry.Descriptor) error {
	var err error
	g := inject.Graph{}

	err = g.Provide(&inject.Object{Value: bus.GetBus()})
	if err != nil {
		return fmt.Errorf("Failed to provide object to the graph: %v", err)
	}
	err = g.Provide(&inject.Object{Value: s.cfg})
	if err != nil {
		return fmt.Errorf("Failed to provide object to the graph: %v", err)
	}
	err = g.Provide(&inject.Object{Value: routing.NewRouteRegister(middleware.RequestMetrics, middleware.RequestTracing)})
	if err != nil {
		return fmt.Errorf("Failed to provide object to the graph: %v", err)
	}
	err = g.Provide(&inject.Object{Value: localcache.New(5*time.Minute, 10*time.Minute)})
	if err != nil {
		return fmt.Errorf("Failed to provide object to the graph: %v", err)
	}

	// Add all services to dependency graph
	for _, service := range services {
		err = g.Provide(&inject.Object{Value: service.Instance})
		if err != nil {
			return fmt.Errorf("Failed to provide object to the graph: %v", err)
		}
	}

	err = g.Provide(&inject.Object{Value: s})
	if err != nil {
		return fmt.Errorf("Failed to provide object to the graph: %v", err)
	}

	// Inject dependencies to services
	if err := g.Populate(); err != nil {
		return fmt.Errorf("Failed to populate service dependency: %v", err)
	}

	return nil
}

func (s *Server) loadConfiguration() {
	err := s.cfg.Load(&setting.CommandLineArgs{
		Config:   s.configFile,
		HomePath: s.homePath,
		Args:     flag.Args(),
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start grafana. error: %s\n", err.Error())
		os.Exit(1)
	}

	s.log.Info(
		"Starting "+setting.ApplicationName,
		"version", version,
		"commit", commit,
		"branch", buildBranch,
		"compiled", time.Unix(setting.BuildStamp, 0),
	)

	s.cfg.LogConfigSources()
}

func (s *Server) Shutdown(reason string) error {
	s.log.Info("Shutdown started", "reason", reason)

	s.shutdownReason = reason
	s.shutdownInProgress = true

	// call cancel func on root context
	s.shutdownFn()

	return s.childRoutines.Wait()
}

func (s *Server) ExitCode(reason error) int {
	code := 1

	if reason == context.Canceled && s.shutdownReason != "" {
		reason = fmt.Errorf(s.shutdownReason)
		code = 0
	}

	s.log.Error("Server shutdown", "reason", reason)

	return code
}

func (s *Server) writePIDFile() {
	if s.pidFile == "" {
		return
	}

	// Ensure the required directory structure exists.
	err := os.MkdirAll(filepath.Dir(s.pidFile), 0700)
	if err != nil {
		s.log.Error("Failed to verify pid directory", "error", err)
		os.Exit(1)
	}

	// Retrieve the PID and write it.
	pid := strconv.Itoa(os.Getpid())
	if err := ioutil.WriteFile(s.pidFile, []byte(pid), 0644); err != nil {
		s.log.Error("Failed to write pidfile", "error", err)
		os.Exit(1)
	}

	s.log.Info("Writing PID file", "path", s.pidFile, "pid", pid)
}

func sendSystemdNotification(state string) error {
	notifySocket := os.Getenv("NOTIFY_SOCKET")

	if notifySocket == "" {
		return fmt.Errorf("NOTIFY_SOCKET environment variable empty or unset")
	}

	socketAddr := &net.UnixAddr{
		Name: notifySocket,
		Net:  "unixgram",
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(state))

	return err
}
