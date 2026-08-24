package obox

import (
	"epos-proxy/internal/logger"

	"github.com/gofiber/fiber/v3"
)

func (m *Module) HandleLanConnection(ctx fiber.Ctx) error {
	logger.Debug("[obox] LAN health check /odoo/")
	if dbURL, _ := m.GetCredentials(); dbURL != "" {
		m.setLANStatus("connected")
		return ctx.JSON(map[string]interface{}{
			"status": "configured",
			"data": map[string]string{
				"serial": m.appID,
				"db_url": dbURL,
			},
		})
	}
	m.setLANStatus("disconnected")
	return ctx.JSON(map[string]interface{}{"status": "not_configured"})
}

func (m *Module) HandleOfflineConnect(ctx fiber.Ctx) error {
	dbURL := ctx.Query("db_url")
	token := ctx.Query("token")
	dbUUID := ctx.Query("db_uuid")

	logger.Debugf("[obox] offline connect received: db_url=%s, db_uuid=%s", dbURL, dbUUID)
	if dbURL == "" || token == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(map[string]interface{}{
			"error": "missing required parameters: db_url and token",
		})
	}

	m.SetCredentials(dbURL, token, dbUUID)
	m.setWsStatus("connecting")
	go m.callOdooOboxConnect(dbURL, token, dbUUID)

	return ctx.SendStatus(fiber.StatusOK)
}

func (m *Module) HandleDisconnect(ctx fiber.Ctx) error {
	logger.Debugf("[obox] /odoo/disconnect request received")
	m.Disconnect()
	return ctx.JSON(map[string]interface{}{"status": "disconnected"})
}
