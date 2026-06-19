// Package sseclient implements a Server-Sent Events consumer for Senda.
// It connects to an SSE endpoint and streams events until ctx is cancelled.
package sseclient

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"senda/internal/model"
	"senda/internal/vars"
)

// SSEEvent is emitted to the caller as events arrive.
type OnEvent func(e model.SSEEvent)

// Connect makes a GET to the SSE endpoint and calls onEvent for every event
// until ctx is cancelled or the connection closes. Returns the full session.
func Connect(ctx context.Context, req model.Request, scope *vars.Scope, onEvent OnEvent) model.SSESession {
	rawURL := scope.Apply(req.URL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return model.SSESession{Error: err.Error()}
	}
	for _, kv := range req.Headers {
		if kv.Enabled {
			httpReq.Header.Set(scope.Apply(kv.Key), scope.Apply(kv.Value))
		}
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return model.SSESession{Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return model.SSESession{Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	var session model.SSESession
	var firstMs int64
	start := time.Now()

	scanner := bufio.NewScanner(resp.Body)
	var (
		id    string
		event string
		data  strings.Builder
	)

	flush := func() {
		if data.Len() == 0 {
			return
		}
		now := time.Now().UnixMilli()
		if firstMs == 0 {
			firstMs = time.Since(start).Milliseconds()
		}
		e := model.SSEEvent{
			ID:    id,
			Event: event,
			Data:  data.String(),
			At:    now,
		}
		session.Events = append(session.Events, e)
		session.Count++
		if onEvent != nil {
			onEvent(e)
		}
		id = ""
		event = ""
		data.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "id:"):
			id = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		case line == "":
			flush()
		}
	}
	flush() // trailing data without final blank line
	session.FirstMs = firstMs
	return session
}
