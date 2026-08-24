package obox

import (
	"net/http"
	"testing"

	"epos-proxy/internal/config"
	"epos-proxy/internal/testutil"

	"github.com/gofiber/fiber/v3"
)

func createTestModule(t *testing.T) (*Module, *fiber.App) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	cfg, err := config.NewManager()
	testutil.ExpectedNoError(t, err)
	app := fiber.New()

	m := NewModule(cfg, 4545)
	t.Cleanup(m.Stop)
	app.Get("/odoo/", m.HandleLanConnection)
	app.Get("/odoo/connect", m.HandleOfflineConnect)
	app.Get("/odoo/disconnect", m.HandleDisconnect)
	return m, app
}

func TestObox_CredentialsAndConnection(t *testing.T) {
	m, _ := createTestModule(t)
	testutil.ExpectedEqual(t, m.GetWebsocketStatus(), "disconnected")

	m.SetCredentials("http://127.0.0.1:8069", "token-xyz", "db-uuid-1")

	dbURL, tok := m.GetCredentials()
	testutil.ExpectedEqual(t, dbURL, "http://127.0.0.1:8069")
	testutil.ExpectedEqual(t, tok, "token-xyz")
	testutil.ExpectedEqual(t, m.GetDbURL(), "http://127.0.0.1:8069")

	m.ClearCredentials()
	testutil.ExpectedEqual(t, m.GetDbURL(), "")
}

func TestObox_StatusChangeListener(t *testing.T) {
	m, _ := createTestModule(t)

	called := false
	m.OnStatusChange(func() {
		called = true
	})

	m.setWsStatus("connected")
	testutil.ExpectedTrue(t, called)
}

func TestObox_LANContact(t *testing.T) {
	m, app := createTestModule(t)
	testutil.ExpectedEqual(t, m.GetLANStatus(), "disconnected")
	m.SetCredentials("http://127.0.0.1:8069", "tok", "uuid-1")

	statusChanged := false
	m.OnStatusChange(func() {
		statusChanged = true
	})

	req, _ := http.NewRequest("GET", "/odoo/", nil)
	_, err := app.Test(req)
	testutil.ExpectedNoError(t, err)
	testutil.ExpectedEqual(t, m.GetLANStatus(), "connected")
	testutil.ExpectedTrue(t, statusChanged)

	statusChanged = false
	req, _ = http.NewRequest("GET", "/odoo/", nil)
	_, err = app.Test(req)
	testutil.ExpectedNoError(t, err)
	testutil.ExpectedEqual(t, m.GetLANStatus(), "connected")
	testutil.ExpectedFalse(t, statusChanged)

	m.Disconnect()
	testutil.ExpectedEqual(t, m.GetLANStatus(), "disconnected")
}
