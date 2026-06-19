package mockserver

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"
)

var proxyClient = &http.Client{Timeout: 30 * time.Second}

// proxyPass forwards the request to target (a base URL) preserving the path,
// query, method, headers and body, then copies the upstream response back to w.
// Returns the upstream status code, or 502 on failure.
func proxyPass(w http.ResponseWriter, r *http.Request, target string, body []byte) int {
	url := strings.TrimRight(target, "/") + r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "mock proxy: "+err.Error(), http.StatusBadGateway)
		return http.StatusBadGateway
	}
	for k, vs := range r.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := proxyClient.Do(req)
	if err != nil {
		http.Error(w, "mock proxy: "+err.Error(), http.StatusBadGateway)
		return http.StatusBadGateway
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	return resp.StatusCode
}
