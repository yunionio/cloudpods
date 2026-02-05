package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
)

func chatHandlerInfo(method, prefix string, handler func(context.Context, http.ResponseWriter, *http.Request)) *appsrv.SHandlerInfo {
	log.Debugf("%s - %s", method, prefix)
	hi := appsrv.SHandlerInfo{}
	hi.SetMethod(method)
	hi.SetPath(prefix)
	hi.SetHandler(handler)
	hi.SetProcessTimeout(6 * time.Hour)
	// Use default worker manager with default pool size (usually 32)
	// instead of uploader worker which has limited pool size (4)
	return &hi
}

func mcpAgentChatStreamHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, _, body := appsrv.FetchEnv(ctx, w, r)
	id := params["<id>"]
	if len(id) == 0 {
		httperrors.MissingParameterError(ctx, w, "id")
		return
	}

	token := AppContextToken(ctx)
	s := auth.GetSession(ctx, token, FetchRegion(r))

	// Prepare request to backend
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	// Forward the request body to the backend
	var bodyReader io.Reader
	if body != nil {
		bodyStr := body.String()
		bodyReader = strings.NewReader(bodyStr)
	}

	path := fmt.Sprintf("/mcp_agents/%s/chat-stream", id)
	resp, err := s.RawVersionRequest(
		modules.MCPAgent.ServiceType(),
		modules.MCPAgent.EndpointType(),
		"POST",
		path,
		headers,
		bodyReader,
	)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, errors.Wrap(err, "request backend"))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Read error body
		respBody, _ := io.ReadAll(resp.Body)
		// Try to parse as JSON error if possible, or just return as is
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			httperrors.InputParameterError(ctx, w, "backend error: %s", string(respBody))
		} else {
			httperrors.GeneralServerError(ctx, w, fmt.Errorf("backend error %d: %s", resp.StatusCode, string(respBody)))
		}
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// For now just standard SSE headers.

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Stream the response from backend to client
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := w.Write(buf[:n]); wErr != nil {
				log.Errorf("write response error: %v", wErr)
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Errorf("read backend response error: %v", err)
			}
			break
		}
	}
}
