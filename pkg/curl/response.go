package curl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Response holds the result of an HTTP request.
type Response struct {
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	BodyBytes []byte            `json:"-"`
	Cookies   []*http.Cookie    `json:"-"`
	FinalURL  string            `json:"final_url,omitempty"`
	Duration  time.Duration     `json:"-"`
}

// responseJSON is the clean JSON representation.
type responseJSON struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Cookies    []cookieJSON      `json:"cookies,omitempty"`
	FinalURL   string            `json:"final_url,omitempty"`
	DurationMs int64             `json:"duration_ms"`
}

type cookieJSON struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"http_only,omitempty"`
}

// MarshalJSON provides a clean JSON representation with duration in ms and simplified cookies.
func (r *Response) MarshalJSON() ([]byte, error) {
	rj := responseJSON{
		Status:     r.Status,
		Headers:    r.Headers,
		Body:       r.Body,
		FinalURL:   r.FinalURL,
		DurationMs: r.Duration.Milliseconds(),
	}
	for _, c := range r.Cookies {
		rj.Cookies = append(rj.Cookies, cookieJSON{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
		})
	}
	return json.Marshal(rj)
}

// Format returns a human-readable representation of the response.
func (r *Response) Format() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "HTTP %d\n\n", r.Status)

	sb.WriteString("--- Response Headers ---\n")
	for k, v := range r.Headers {
		fmt.Fprintf(&sb, "%s: %s\n", k, v)
	}

	sb.WriteString("\n--- Response Body ---\n")
	sb.WriteString(r.Body)

	if r.FinalURL != "" {
		fmt.Fprintf(&sb, "\n\n--- Final URL ---\n%s", r.FinalURL)
	}

	fmt.Fprintf(&sb, "\n\n--- Duration ---\n%dms", r.Duration.Milliseconds())

	return sb.String()
}

// JSON returns the response serialized as JSON with proper formatting.
func (r *Response) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// JSONSmart returns JSON with the body parsed as a JSON object when Content-Type is application/json.
// This makes the output jq-friendly.
func (r *Response) JSONSmart() ([]byte, error) {
	ct := r.Headers["Content-Type"]
	if ct == "" {
		ct = r.Headers["content-type"]
	}
	isJSON := strings.Contains(ct, "application/json")

	if !isJSON || r.Body == "" {
		return json.MarshalIndent(r, "", "  ")
	}

	// Parse body into raw JSON so it's not double-escaped
	var parsedBody json.RawMessage
	if err := json.Unmarshal([]byte(r.Body), &parsedBody); err != nil {
		// If body isn't valid JSON, fall back to string
		return json.MarshalIndent(r, "", "  ")
	}

	rj := smartResponseJSON{
		Status:     r.Status,
		Headers:    r.Headers,
		Body:       parsedBody,
		FinalURL:   r.FinalURL,
		DurationMs: r.Duration.Milliseconds(),
	}
	for _, c := range r.Cookies {
		rj.Cookies = append(rj.Cookies, cookieJSON{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
		})
	}
	return json.MarshalIndent(rj, "", "  ")
}

type smartResponseJSON struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
	Cookies    []cookieJSON      `json:"cookies,omitempty"`
	FinalURL   string            `json:"final_url,omitempty"`
	DurationMs int64             `json:"duration_ms"`
}

// BatchResult holds the result of a single request within a batch.
type BatchResult struct {
	Index    int       `json:"index"`
	URL      string    `json:"url"`
	Response *Response `json:"response,omitempty"`
	Error    string    `json:"error,omitempty"`
	CurlCmd  string    `json:"curl_cmd,omitempty"`
}
