package curl

import (
	"fmt"
	"sort"
	"strings"
)

// ToCurl converts RequestOptions to an equivalent curl command string.
func ToCurl(opts RequestOptions) string {
	var parts []string
	parts = append(parts, "curl")

	method := strings.ToUpper(opts.Method)
	if method == "" {
		method = "GET"
	}
	if method != "GET" {
		parts = append(parts, fmt.Sprintf("-X %s", method))
	}

	parts = append(parts, fmt.Sprintf("'%s'", opts.URL))

	// Headers in deterministic order
	if len(opts.Headers) > 0 {
		keys := make([]string, 0, len(opts.Headers))
		for k := range opts.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("-H '%s: %s'", k, opts.Headers[k]))
		}
	}

	if opts.UserAgent != "" {
		parts = append(parts, fmt.Sprintf("-A '%s'", opts.UserAgent))
	}

	if opts.Body != "" {
		parts = append(parts, fmt.Sprintf("-d '%s'", opts.Body))
	}

	if len(opts.Cookies) > 0 {
		var cookieParts []string
		for _, c := range opts.Cookies {
			cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
		parts = append(parts, fmt.Sprintf("-b '%s'", strings.Join(cookieParts, "; ")))
	}

	if opts.Timeout > 0 {
		parts = append(parts, fmt.Sprintf("--connect-timeout %d", opts.Timeout))
	}

	if opts.Proxy != "" {
		parts = append(parts, fmt.Sprintf("--proxy '%s'", opts.Proxy))
	}

	if opts.InsecureSkipVerify {
		parts = append(parts, "--insecure")
	}

	if opts.DisableRedirect {
		parts = append(parts, "--max-redirs 0")
	}

	if opts.ForceHTTP1 {
		parts = append(parts, "--http1.1")
	}

	if opts.ForceHTTP3 {
		parts = append(parts, "--http3")
	}

	return strings.Join(parts, " \\\n  ")
}
