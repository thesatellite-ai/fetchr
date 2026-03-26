package curl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Logger logs HTTP request/response pairs.
type Logger interface {
	Log(ctx context.Context, entry LogEntry) error
}

// LogEntry represents a single logged request/response pair.
type LogEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Request   RequestOptions `json:"request"`
	Response  *Response      `json:"response,omitempty"`
	Error     string         `json:"error,omitempty"`
	Duration  time.Duration  `json:"duration"`
	CurlCmd   string         `json:"curl_cmd,omitempty"`
}

// FileLogger appends JSONL entries to a file.
type FileLogger struct {
	path string
	mu   sync.Mutex
}

// NewFileLogger creates a FileLogger that writes to the given path.
func NewFileLogger(path string) *FileLogger {
	return &FileLogger{path: path}
}

func (l *FileLogger) Log(_ context.Context, entry LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal log entry: %w", err)
	}
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// WebhookLogger POSTs LogEntry JSON to a URL (fire-and-forget).
type WebhookLogger struct {
	url    string
	client *http.Client
}

// NewWebhookLogger creates a WebhookLogger that posts to the given URL.
func NewWebhookLogger(url string) *WebhookLogger {
	return &WebhookLogger{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (l *WebhookLogger) Log(_ context.Context, entry LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return nil // swallow marshal errors for fire-and-forget
	}

	go func() {
		resp, err := l.client.Post(l.url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "webhook logger error: %v\n", err)
			return
		}
		resp.Body.Close()
	}()

	return nil
}

// MultiLogger fans out to multiple loggers.
type MultiLogger struct {
	loggers []Logger
}

// NewMultiLogger creates a MultiLogger that writes to all given loggers.
func NewMultiLogger(loggers ...Logger) *MultiLogger {
	return &MultiLogger{loggers: loggers}
}

func (l *MultiLogger) Log(ctx context.Context, entry LogEntry) error {
	for _, logger := range l.loggers {
		if err := logger.Log(ctx, entry); err != nil {
			fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		}
	}
	return nil
}
