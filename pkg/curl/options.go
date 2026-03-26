package curl

// RequestOptions holds all configurable fields for an HTTP request.
// Every CycleTLS option is exposed here.
type RequestOptions struct {
	// Core
	URL    string `json:"url" jsonschema:"required,description=The URL to request"`
	Method string `json:"method,omitempty" jsonschema:"description=HTTP method. Default: GET"`

	// Headers & Body
	Headers     map[string]string `json:"headers,omitempty"`
	HeaderOrder []string          `json:"header_order,omitempty"`
	Body        string            `json:"body,omitempty"`
	Cookies     []Cookie          `json:"cookies,omitempty"`

	// TLS Fingerprinting
	Ja3              string `json:"ja3,omitempty"`
	Ja4r             string `json:"ja4r,omitempty"`
	HTTP2Fingerprint string `json:"http2_fingerprint,omitempty"`
	QUICFingerprint  string `json:"quic_fingerprint,omitempty"`
	UserAgent        string `json:"user_agent,omitempty"`

	// Connection
	Proxy              string `json:"proxy,omitempty"`
	Timeout            int    `json:"timeout,omitempty"`
	ServerName         string `json:"server_name,omitempty"`
	InsecureSkipVerify bool   `json:"insecure,omitempty"`

	// Protocol
	ForceHTTP1 bool   `json:"force_http1,omitempty"`
	ForceHTTP3 bool   `json:"force_http3,omitempty"`
	Protocol   string `json:"protocol,omitempty"` // http1, http2, http3

	// Behavior
	DisableRedirect       bool `json:"disable_redirect,omitempty"`
	DisableGrease         bool `json:"disable_grease,omitempty"`
	TLS13AutoRetry        bool `json:"tls13_auto_retry,omitempty"`
	EnableConnectionReuse bool `json:"enable_connection_reuse,omitempty"`

	// Logging override per-request
	PrintCurl bool `json:"print_curl,omitempty"`

	// Profile name (resolved from config)
	Profile string `json:"profile,omitempty"`
}

// Cookie represents an HTTP cookie to send with a request.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Merge returns a new RequestOptions with defaults filled in from base.
// Fields set in opts take precedence over base.
func Merge(base, opts RequestOptions) RequestOptions {
	result := base

	if opts.URL != "" {
		result.URL = opts.URL
	}
	if opts.Method != "" {
		result.Method = opts.Method
	}
	if opts.Headers != nil {
		if result.Headers == nil {
			result.Headers = make(map[string]string)
		}
		for k, v := range opts.Headers {
			result.Headers[k] = v
		}
	}
	if opts.HeaderOrder != nil {
		result.HeaderOrder = opts.HeaderOrder
	}
	if opts.Body != "" {
		result.Body = opts.Body
	}
	if opts.Cookies != nil {
		result.Cookies = opts.Cookies
	}
	if opts.Ja3 != "" {
		result.Ja3 = opts.Ja3
	}
	if opts.Ja4r != "" {
		result.Ja4r = opts.Ja4r
	}
	if opts.HTTP2Fingerprint != "" {
		result.HTTP2Fingerprint = opts.HTTP2Fingerprint
	}
	if opts.QUICFingerprint != "" {
		result.QUICFingerprint = opts.QUICFingerprint
	}
	if opts.UserAgent != "" {
		result.UserAgent = opts.UserAgent
	}
	if opts.Proxy != "" {
		result.Proxy = opts.Proxy
	}
	if opts.Timeout != 0 {
		result.Timeout = opts.Timeout
	}
	if opts.ServerName != "" {
		result.ServerName = opts.ServerName
	}
	if opts.InsecureSkipVerify {
		result.InsecureSkipVerify = true
	}
	if opts.ForceHTTP1 {
		result.ForceHTTP1 = true
	}
	if opts.ForceHTTP3 {
		result.ForceHTTP3 = true
	}
	if opts.Protocol != "" {
		result.Protocol = opts.Protocol
	}
	if opts.DisableRedirect {
		result.DisableRedirect = true
	}
	if opts.DisableGrease {
		result.DisableGrease = true
	}
	if opts.TLS13AutoRetry {
		result.TLS13AutoRetry = true
	}
	if opts.EnableConnectionReuse {
		result.EnableConnectionReuse = true
	}
	if opts.PrintCurl {
		result.PrintCurl = true
	}
	if opts.Profile != "" {
		result.Profile = opts.Profile
	}

	return result
}
