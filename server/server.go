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
	app := fiber.New(fiber.Config{
		AppName: "ePOS proxy",
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins:        []string{"*"},
		AllowPrivateNetwork: true,
	}))

	app.Post("/p/:printerId/cgi-bin/epos/service.cgi", func(ctx fiber.Ctx) error {
		printerId := ctx.Params("printerId")
		logger.Debugf("Print request received for printer: %s", printerId)
		return printData(mgr, ctx, printer.RawPrinter{Category: printer.PrinterThermal, PrinterIp: printerId})
	})

	app.Post("/cgi-bin/epos/service.cgi", func(ctx fiber.Ctx) error {
		logger.Debugf("Print request received (auto printer selection)")
		return printData(mgr, ctx, printer.RawPrinter{Category: printer.PrinterOffice, PrinterIp: ""})
	})

	app.Post("/p/:printerId/print/pdf", func(ctx fiber.Ctx) error {
		printerId := ctx.Params("printerId")
		logger.Debugf("Print request received for printer: %s", printerId)
		return printPDF(mgr, ctx, printer.RawPrinter{Category: printer.PrinterOffice, PrinterIp: printerId})
	})

	server := &Server{app: app, Port: port}
	server.running.Store(true)
	go func() {
		logger.Infof("HTTP server listening on 0.0.0.0:%d", port)
		err := app.Listen(fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			logger.Error("EPOS Server Error: ", err)
		}
		server.running.Store(false)
		logger.Warn("HTTP server stopped")
	}()
	return server
}

func printData(mgr *printer.Manager, ctx fiber.Ctx, rawPrinter printer.RawPrinter) error {
	logger.Debugf("Processing print job for printer: %v", rawPrinter)
	if len(ctx.Body()) == 0 {
		logger.Warnf("Received empty EPOS payload for printer: %v", rawPrinter.PrinterIp)
		return ctx.Status(400).SendString("Empty XML payload")
	}
	jobData, err := escpos.ParseXML(ctx.Body())
	if err != nil {
		logger.Errorf("XML parsing error: %v", err)
		return ctx.XML(EPOSResponse{Success: false, Code: "SchemaError", Status: ""})
	}
	logger.Debug("XML parsed successfully")

	reply, err := mgr.WriteAsync(jobData, rawPrinter)
	if err == nil {
		logger.Debug("Print job queued")
		result := <-reply
		if !result.OK {
			err = result.Err
		}
	}
	if err != nil {
		retCode := ""
		if errors.Is(err, printer.ErrQueueFull) {
			retCode = "TooManyRequests"
			logger.Warn("Printer queue full")
		} else {
			retCode = "EX_BADPORT"
		}
		logger.Errorf("Print error [%s]: %v, Printer: %v", retCode, err, rawPrinter)
		return ctx.XML(EPOSResponse{Success: false, Code: retCode, Status: ""})
	}
	logger.Debugf("Print job completed successfully for printer: %v", rawPrinter)
	return ctx.XML(EPOSResponse{Success: true, Code: "", Status: ""})
}

func printPDF(mgr *printer.Manager, ctx fiber.Ctx, rawPrinter printer.RawPrinter) error {
	logger.Debugf("Processing PDF print job for printer: %v", rawPrinter)
	pdfBytes := ctx.Body()
	if len(pdfBytes) == 0 {
		logger.Warnf("Received empty PDF payload for printer: %v", rawPrinter)
		return ctx.Status(400).SendString("Empty PDF payload")
	}
	logger.Debugf("Received PDF data for printer: %v", rawPrinter)
	reply, err := mgr.WriteAsync(pdfBytes, rawPrinter)
	if err == nil {
		logger.Debug("Print job queued")
		result := <-reply
		if !result.OK {
			err = result.Err
		}
	}
	if err != nil {
		logger.Errorf("Print PDF Error for printer %v: %v", rawPrinter, err)
	}
	return err
}

func (s *Server) Stop() error {
	logger.Infof("Stopping HTTP server")
	return s.app.Shutdown()
}

func (s *Server) Running() bool {
	return s.running.Load()
}
