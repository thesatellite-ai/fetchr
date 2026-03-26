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

// BatchResult holds the result of a single request within a batch.
type BatchResult struct {
	Index    int       `json:"index"`
	URL      string    `json:"url"`
	Response *Response `json:"response,omitempty"`
	Error    string    `json:"error,omitempty"`
	CurlCmd  string    `json:"curl_cmd,omitempty"`
}
