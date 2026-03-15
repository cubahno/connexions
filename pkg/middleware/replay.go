package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
)

// headerReplayMatch is the HTTP header that activates replay.
// When present (even with empty value), replay middleware is activated.
// If the header contains comma-separated dotted paths, those override configured match fields.
//
// Example: X-Cxs-Replay: data.name,data.address.zip
const headerReplayMatch = "X-Cxs-Replay"

// ReplayRecord holds a recorded response along with request metadata for debugging.
//
// Data is the response body bytes.
// Headers are the response headers (excluding internal X-Cxs-* headers).
// StatusCode is the HTTP status code of the response.
// ContentType is the Content-Type header of the response.
// IsFromUpstream is true if the response came from an upstream service.
// RequestBody is the original request body (stored for debugging).
// MatchValues maps field paths to their extracted values (stored for debugging).
// CreatedAt is when the recording was created.
type ReplayRecord struct {
	Data           []byte            `json:"data"`
	Headers        map[string]string `json:"headers"`
	StatusCode     int               `json:"statusCode"`
	ContentType    string            `json:"contentType"`
	IsFromUpstream bool              `json:"isFromUpstream"`
	RequestBody    []byte            `json:"requestBody"`
	MatchValues    map[string]any    `json:"matchValues"`
	CreatedAt      time.Time         `json:"createdAt"`
}

// parseReplayHeader parses the X-Cxs-Replay header value into match fields.
// Returns nil if the header value is empty.
func parseReplayHeader(headerValue string) []string {
	if headerValue == "" {
		return nil
	}
	fields := strings.Split(headerValue, ",")
	result := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			result = append(result, f)
		}
	}
	return result
}

// resolveReplayParams resolves the match fields and pattern path for a replay request.
// Activation rules:
//   - If X-Cxs-Replay header is present → always activate (header value overrides config match fields)
//   - If auto-replay is true in config → activate for configured endpoints without header
//   - Otherwise → skip
func resolveReplayParams(req *http.Request, cfg *config.ServiceConfig) (matchFields []string, patternPath string) {
	// Check header presence (not just value — empty header is valid)
	headerValues, headerPresent := req.Header[http.CanonicalHeaderKey(headerReplayMatch)]

	var headerValue string
	if headerPresent && len(headerValues) > 0 {
		headerValue = headerValues[0]
	}

	// Check if auto-replay is enabled
	autoReplay := cfg.Cache != nil && cfg.Cache.Replay != nil && cfg.Cache.Replay.AutoReplay

	// No header and no auto-replay → skip
	if !headerPresent && !autoReplay {
		return nil, ""
	}

	endpointPath := getEndpointPath(req, cfg.Name)

	// Try to get pattern path and config match fields from replay config
	var configMatch []string
	if cfg.Cache != nil && cfg.Cache.Replay != nil {
		pattern, ep := cfg.Cache.Replay.GetEndpoint(endpointPath, req.Method)
		if ep != nil {
			patternPath = pattern
			configMatch = ep.Match
		}
	}

	// auto-replay without header requires a configured endpoint
	if !headerPresent && len(configMatch) == 0 {
		return nil, ""
	}

	// If no pattern matched, use the actual endpoint path
	if patternPath == "" {
		patternPath = endpointPath
	}

	// Header value overrides config match fields
	if fields := parseReplayHeader(headerValue); len(fields) > 0 {
		return fields, patternPath
	}

	// Header present but empty value, or auto-replay → fall back to config
	return configMatch, patternPath
}

// buildReplayKey builds a deterministic key from method, pattern path, match fields, and request body.
// Returns a SHA-256 hex digest.
func buildReplayKey(method, patternPath string, matchFields []string, body []byte) string {
	sorted := make([]string, len(matchFields))
	copy(sorted, matchFields)
	sort.Strings(sorted)

	var sb strings.Builder
	sb.WriteString(method)
	sb.WriteString(":")
	sb.WriteString(patternPath)

	for _, field := range sorted {
		val := extractJSONPath(body, field)
		sb.WriteString("|")
		sb.WriteString(field)
		sb.WriteString("=")
		sb.WriteString(formatValue(val))
	}

	hash := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("%x", hash)
}

// readAndRestoreBody reads the request body and restores it so subsequent handlers can read it.
func readAndRestoreBody(req *http.Request) []byte {
	if req.Body == nil {
		return nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

// getEndpointPath strips the service name prefix from the request URL path.
func getEndpointPath(req *http.Request, serviceName string) string {
	prefix := "/" + serviceName
	path := strings.TrimPrefix(req.URL.Path, prefix)
	if path == "" {
		path = "/"
	}
	return path
}

// deserializeReplayRecord converts a value retrieved from the DB table into a ReplayRecord.
// Handles both direct *ReplayRecord (memory backend) and map[string]any (Redis backend).
func deserializeReplayRecord(val any) *ReplayRecord {
	if val == nil {
		return nil
	}

	// Direct type — memory backend stores it as-is
	if rec, ok := val.(*ReplayRecord); ok {
		return rec
	}

	// Redis backend returns map[string]any — re-serialize and parse
	data, err := json.Marshal(val)
	if err != nil {
		return nil
	}

	var rec ReplayRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil
	}
	return &rec
}
