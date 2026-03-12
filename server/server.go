package server

import (
	"encoding/xml"
	"errors"
	"fmt"
	"sync/atomic"

	"epos-proxy/escpos"
	"epos-proxy/logger"
	"epos-proxy/printer"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

type EPOSResponse struct {
	XMLName xml.Name `xml:"response"`
	Success bool     `xml:"success,attr"`
	Code    string   `xml:"code,attr"`
	Status  string   `xml:"status,attr"`
}

type Server struct {
	app     *fiber.App
	Port    int
	running atomic.Bool
}

func New(port int, mgr *printer.Manager) *Server {
	logger.Log.Infof("Starting ePOS HTTP server on port %d", port)
	app := fiber.New(fiber.Config{
		AppName: "ePOS proxy",
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins:        []string{"*"},
		AllowPrivateNetwork: true,
	}))

	app.Post("/p/:printerId/cgi-bin/epos/service.cgi", func(ctx fiber.Ctx) error {
		printerId := ctx.Params("printerId")
		logger.Log.Debugf("Print request received for printer: %s", printerId)
		return printData(mgr, ctx, printerId)
	})

	app.Post("/cgi-bin/epos/service.cgi", func(ctx fiber.Ctx) error {
		logger.Log.Debugf("Print request received (auto printer selection)")
		return printData(mgr, ctx, "")
	})

	server := &Server{app: app, Port: port}
	server.running.Store(true)
	go func() {
		logger.Log.Infof("HTTP server listening on 0.0.0.0:%d", port)
		err := app.Listen(fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			logger.Log.Error("EPOS Server Error: ", err)
		}
		server.running.Store(false)
		logger.Log.Warn("HTTP server stopped")
	}()
	return server
}

func printData(mgr *printer.Manager, ctx fiber.Ctx, printerID string) error {
	logger.Log.Debugf("Processing print job for printer: %s", printerID)
	jobData, err := escpos.ParseXML(ctx.Body())
	if err != nil {
		logger.Log.Errorf("XML parse error: %v", err)
		return ctx.XML(EPOSResponse{Success: false, Code: "SchemaError", Status: ""})
	}
	logger.Log.Debug("XML parsed successfully")

	reply, err := mgr.WriteAsync(printerID, jobData)
	if err == nil {
		logger.Log.Debug("Print job queued")
		result := <-reply
		if !result.OK {
			err = result.Err
		}
	}
	if err != nil {
		retCode := ""
		if errors.Is(err, printer.ErrQueueFull) {
			retCode = "TooManyRequests"
			logger.Log.Warn("Printer queue full")
		} else {
			retCode = "EX_BADPORT"
		}
		logger.Log.Errorf("Print error [%s]: %v", retCode, err)
		return ctx.XML(EPOSResponse{Success: false, Code: retCode, Status: ""})
	}
	logger.Log.Infof("Print job completed successfully")
	return ctx.XML(EPOSResponse{Success: true, Code: "", Status: ""})
}

func (s *Server) Stop() error {
	logger.Log.Infof("Stopping HTTP server")
	return s.app.Shutdown()
}

func (s *Server) Running() bool {
	return s.running.Load()
}
