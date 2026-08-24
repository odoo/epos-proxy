package server

import (
	"fmt"
	"sync/atomic"

	"epos-proxy/internal/logger"
	"epos-proxy/internal/obox"
	"epos-proxy/internal/printer"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

type Server struct {
	app     *fiber.App
	Port    int
	running atomic.Bool
	obox    *obox.Module
	mgr     *printer.Manager
}

func New(port int, mgr *printer.Manager, oboxMod *obox.Module) *Server {
	app := fiber.New(fiber.Config{AppName: "Obox app"})
	app.Use(cors.New(cors.Config{
		AllowOrigins:        []string{"*"},
		AllowPrivateNetwork: true,
	}))

	server := &Server{
		app:  app,
		Port: port,
		obox: oboxMod,
		mgr:  mgr,
	}

	server.registerRoutes()
	server.running.Store(true)
	go func() {
		logger.Infof("HTTP server listening on 0.0.0.0:%d", port)
		err := app.Listen(fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			logger.Error("Obox app Server Error: ", err)
		}
		server.running.Store(false)
		logger.Warn("HTTP server stopped")
	}()
	return server
}

func (s *Server) Stop() error {
	logger.Infof("Stopping HTTP server")
	s.running.Store(false)
	return s.app.Shutdown()
}

func (s *Server) Running() bool {
	return s.running.Load()
}

func (s *Server) registerRoutes() {
	registerEPOSRoutes(s)
	registerOboxRoutes(s)
}
