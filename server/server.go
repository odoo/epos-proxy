package server

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"sync/atomic"

	"epos-proxy/escpos"
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
		return printData(mgr, ctx, printerId)
	})

	app.Post("/cgi-bin/epos/service.cgi", func(ctx fiber.Ctx) error {
		return printData(mgr, ctx, "")
	})

	server := &Server{app: app, Port: port}
	server.running.Store(true)
	go func() {
		err := app.Listen(fmt.Sprintf("0.0.0.0:%d", port))
		log.Println("EPOS Server Error:", err)
		server.running.Store(false)
	}()
	return server
}

func printData(mgr *printer.Manager, ctx fiber.Ctx, printerID string) error {
	jobData, err := escpos.ParseXML(ctx.Body())
	if err != nil {
		log.Println("XML Error:", err)
		return ctx.XML(EPOSResponse{Success: false, Code: "SchemaError", Status: ""})
	}

	reply, err := mgr.WriteAsync(printerID, jobData)
	if err == nil {
		result := <-reply
		if !result.OK {
			err = result.Err
		}
	}
	if err != nil {
		retCode := ""
		if errors.Is(err, printer.ErrQueueFull) {
			retCode = "TooManyRequests"
		} else {
			retCode = "EX_BADPORT"
		}
		log.Println("Print Error:", retCode, err)
		return ctx.XML(EPOSResponse{Success: false, Code: retCode, Status: ""})
	}
	return ctx.XML(EPOSResponse{Success: true, Code: "", Status: ""})
}

func (s *Server) Stop() error {
	return s.app.Shutdown()
}

func (s *Server) Running() bool {
	return s.running.Load()
}
