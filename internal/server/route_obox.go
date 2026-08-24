package server

import (
	"epos-proxy/internal/logger"

	"github.com/gofiber/fiber/v3"
)

func registerOboxRoutes(s *Server) {
	s.app.Get("/odoo/", s.obox.HandleLanConnection)
	s.app.Get("/odoo/connect", s.obox.HandleOfflineConnect)
	s.app.Get("/odoo/disconnect", s.obox.HandleDisconnect)

	s.app.Post("/usb/v1/printer/:printerId/cgi-bin/epos/service.cgi", func(ctx fiber.Ctx) error {
		printerId := ctx.Params("printerId")
		logger.Debugf("Obox ePOS print request received for printer: %s", printerId)
		return printReceipt(s.mgr, ctx, printerId)
	})

	s.app.Post("/usb/v1/printer/:printerId/pstprnt", func(ctx fiber.Ctx) error {
		printerId := ctx.Params("printerId")
		logger.Debugf("Obox label print request received for printer: %s", printerId)
		return printLabel(s.mgr, ctx, printerId)
	})
}
