package obox

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"epos-proxy/internal/config"
	"epos-proxy/internal/testutil"
)

func TestObox_BuildDeviceList(t *testing.T) {
	m, _ := createTestModule(t)
	err := m.cfg.AddLanEposPrinter("192.168.1.100")
	testutil.ExpectedNoError(t, err)

	for _, dev := range m.buildDeviceList() {
		testutil.ExpectedEqual(t, dev["type"], "printer")
		testutil.ExpectedEqual(t, dev["name"], "Network - 192.168.1.100")
		testutil.ExpectedEqual(t, dev["identifier"], "ipp_bDoxOTIuMTY4LjEuMTAw")
	}
}

func TestObox_DispatchLocalAction(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var receivedMethod string
	var receivedBody map[string]interface{}

	mockLocal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"result": "dispatched_ok"})
	}))
	defer mockLocal.Close()

	cfg, err := config.NewManager()
	testutil.ExpectedNoError(t, err)

	mockPort := mockLocal.Listener.Addr().(*net.TCPAddr).Port
	m := NewModule(cfg, mockPort)
	t.Cleanup(m.Stop)

	// 1. POST dispatch
	resPost, _ := m.dispatchLocalAction(m.ctx, ActionPayload{
		URL:     "/test-post",
		Method:  "POST",
		Payload: json.RawMessage(`{"test_key":"test_val"}`),
	})
	testutil.ExpectedEqual(t, receivedMethod, "POST")
	testutil.ExpectedEqual(t, receivedBody["test_key"], "test_val")
	resMap, ok := resPost.(map[string]interface{})
	testutil.ExpectedTrue(t, ok)
	testutil.ExpectedEqual(t, resMap["result"], "dispatched_ok")

	// 2. GET dispatch
	resGet, _ := m.dispatchLocalAction(m.ctx, ActionPayload{URL: "/test-get", Method: "GET"})
	testutil.ExpectedEqual(t, receivedMethod, "GET")
	resMapGet, ok := resGet.(map[string]interface{})
	testutil.ExpectedTrue(t, ok)
	testutil.ExpectedEqual(t, resMapGet["result"], "dispatched_ok")
}
