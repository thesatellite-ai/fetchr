package curl

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
)

// Client wraps CycleTLS with a high-level API.
type Client struct {
	cycleClient cycletls.CycleTLS
	defaults    RequestOptions
	logger      Logger
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithDefaults sets default RequestOptions applied to every request.
func WithDefaults(opts RequestOptions) ClientOption {
	return func(c *Client) {
		c.defaults = opts
	}
}

// WithLogger attaches a Logger to the client.
func WithLogger(l Logger) ClientOption {
	return func(c *Client) {
		c.logger = l
	}
}

// New creates a new Client with the given options.
func New(opts ...ClientOption) *Client {
	c := &Client{
		cycleClient: cycletls.Init(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Do executes a single HTTP request.
func (c *Client) Do(ctx context.Context, opts RequestOptions) (*Response, error) {
	merged := Merge(c.defaults, opts)

	if merged.URL == "" {
		return nil, fmt.Errorf("url is required")
	}

	method := strings.ToUpper(merged.Method)
	if method == "" {
		method = "GET"
	}

	cycleOpts := toCycleOptions(merged)

	start := time.Now()
	resp, err := c.cycleClient.Do(merged.URL, cycleOpts, method)
	duration := time.Since(start)

	if c.logger != nil {
		entry := LogEntry{
			Timestamp: start,
			Request:   merged,
			Duration:  duration,
		}
		if merged.PrintCurl {
			entry.CurlCmd = ToCurl(merged)
		}
		if err != nil {
			entry.Error = err.Error()
		} else {
			entry.Response = cycleResponseToResponse(resp, duration)
		}
		_ = c.logger.Log(ctx, entry)
	}

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return cycleResponseToResponse(resp, duration), nil
}

// Batch executes multiple HTTP requests concurrently.
func (c *Client) Batch(ctx context.Context, requests []RequestOptions) ([]*BatchResult, error) {
	results := make([]*BatchResult, len(requests))
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		go func(i int, req RequestOptions) {
			defer wg.Done()
			result := &BatchResult{
				Index: i,
				URL:   req.URL,
			}
			if req.PrintCurl {
				merged := Merge(c.defaults, req)
				result.CurlCmd = ToCurl(merged)
			}
			resp, err := c.Do(ctx, req)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.Response = resp
			}
			results[i] = result
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

// Close shuts down the CycleTLS client.
func (c *Client) Close() {
	c.cycleClient.Close()
}

func toCycleOptions(opts RequestOptions) cycletls.Options {
	co := cycletls.Options{
		Headers:               opts.Headers,
		HeaderOrder:           opts.HeaderOrder,
		Body:                  opts.Body,
		Ja3:                   opts.Ja3,
		Ja4r:                  opts.Ja4r,
		HTTP2Fingerprint:      opts.HTTP2Fingerprint,
		QUICFingerprint:       opts.QUICFingerprint,
		UserAgent:             opts.UserAgent,
		Proxy:                 opts.Proxy,
		Timeout:               opts.Timeout,
		ServerName:            opts.ServerName,
		InsecureSkipVerify:    opts.InsecureSkipVerify,
		ForceHTTP1:            opts.ForceHTTP1,
		ForceHTTP3:            opts.ForceHTTP3,
		Protocol:              opts.Protocol,
		DisableRedirect:       opts.DisableRedirect,
		DisableGrease:         opts.DisableGrease,
		TLS13AutoRetry:        opts.TLS13AutoRetry,
		EnableConnectionReuse: opts.EnableConnectionReuse,
	}

	if len(opts.Cookies) > 0 {
		cycleCookies := make([]cycletls.Cookie, len(opts.Cookies))
		for i, c := range opts.Cookies {
			cycleCookies[i] = cycletls.Cookie{
				Name:  c.Name,
				Value: c.Value,
			}
		}
		co.Cookies = cycleCookies
	}

	return co
}

func cycleResponseToResponse(resp cycletls.Response, duration time.Duration) *Response {
	var cookies []*http.Cookie
	for _, c := range resp.Cookies {
		cookies = append(cookies, c)
	}

	return &Response{
		Status:    resp.Status,
		Headers:   resp.Headers,
		Body:      resp.Body,
		BodyBytes: resp.BodyBytes,
		Cookies:   cookies,
		FinalURL:  resp.FinalUrl,
		Duration:  duration,
	}
}
