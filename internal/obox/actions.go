package obox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"epos-proxy/internal/logger"
	"epos-proxy/internal/printer"
	"epos-proxy/internal/util"
)

func (m *Module) buildDeviceList() []map[string]string {
	discovered := printer.DiscoverAllPrinters(m.cfg)
	list := make([]map[string]string, 0, len(discovered.Printers))
	for _, p := range discovered.Printers {
		list = append(list, map[string]string{
			"name":       p.Name,
			"identifier": p.Identifier,
			"type":       "printer",
		})
	}
	return list
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func buildPostBody(payload json.RawMessage) io.Reader {
	if len(payload) == 0 || string(payload) == "null" {
		return nil
	}

	var str string
	if json.Unmarshal(payload, &str) == nil {
		return strings.NewReader(str)
	}

	return bytes.NewReader(payload)
}

func (m *Module) dispatchLocalAction(ctx context.Context, actionPayload ActionPayload) (interface{}, error) {
	targetURL := actionPayload.URL
	if strings.HasPrefix(targetURL, "/") {
		targetURL = fmt.Sprintf("http://%s%s", util.LocalAddr(m.port), targetURL)
	} else if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = fmt.Sprintf("http://%s", targetURL)
	}

	method := actionPayload.Method
	if method == "" {
		method = "GET"
	}

	body := buildPostBody(actionPayload.Payload)

	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("[obox queue] build request error: %v", err)
	}

	for k, v := range actionPayload.Headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[obox queue] local action HTTP error url=%s: %v", targetURL, err)
	}
	defer resp.Body.Close()

	const maxResponseSize = 5 << 20 // 5 MiB
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("local service returned status %d: %s", resp.StatusCode, string(raw))
	}

	var result interface{}
	if err := json.Unmarshal(raw, &result); err == nil {
		logger.Debugf("[obox queue] local action result: %v", result)
		return result, nil
	}

	logger.Debugf("[obox queue] local action raw result: %s", string(raw))
	return string(raw), nil
}
