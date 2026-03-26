package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thesatellite-ai/fetchr/internal/config"
	mcpserver "github.com/thesatellite-ai/fetchr/internal/mcp"
	"github.com/thesatellite-ai/fetchr/internal/version"
	"github.com/thesatellite-ai/fetchr/pkg/curl"
)

var configPath string

func main() {
	rootCmd := &cobra.Command{
		Use:   "fetchr",
		Short: "HTTP client with TLS fingerprinting — CLI, MCP server, and Go package",
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config.jsonc (overrides auto-discovery)")

	rootCmd.AddCommand(
		serveCmd(),
		requestCmd(),
		batchCmd(),
		configCmd(),
		versionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadConfig() *config.Config {
	if configPath != "" {
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("Failed to load config %s: %v", configPath, err)
		}
		return cfg
	}
	cfg, err := config.LoadConfigAuto()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return cfg
}

// --- serve ---

func serveCmd() *cobra.Command {
	var transport, port string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server (stdio or SSE)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			server := mcpserver.NewServer(cfg)
			defer server.Close()

			switch transport {
			case "sse":
				handler := server.SSEHandler()
				log.Printf("fetchr SSE server listening on %s", port)
				return http.ListenAndServe(port, corsMiddleware(handler))
			default:
				return server.RunStdio(context.Background())
			}
		},
	}

	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport mode: stdio or sse")
	cmd.Flags().StringVar(&port, "port", ":8080", "Port to listen on (SSE mode only)")

	return cmd
}

// --- request ---

func requestCmd() *cobra.Command {
	var (
		method      string
		headers     []string
		data        string
		ja3         string
		ja4r        string
		h2fp        string
		userAgent   string
		proxy       string
		timeout     int
		insecure    bool
		noRedirect  bool
		http1       bool
		http3       bool
		profile     string
		printCurl   bool
		output      string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "request <url>",
		Short: "Make a single HTTP request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()

			opts := curl.RequestOptions{
				URL:                args[0],
				Method:             method,
				Body:               data,
				Ja3:                ja3,
				Ja4r:               ja4r,
				HTTP2Fingerprint:   h2fp,
				UserAgent:          userAgent,
				Proxy:              proxy,
				Timeout:            timeout,
				InsecureSkipVerify: insecure,
				DisableRedirect:    noRedirect,
				ForceHTTP1:         http1,
				ForceHTTP3:         http3,
				Profile:            profile,
				PrintCurl:          printCurl,
			}

			if len(headers) > 0 {
				opts.Headers = make(map[string]string)
				for _, h := range headers {
					k, v, ok := strings.Cut(h, ":")
					if !ok {
						return fmt.Errorf("invalid header format %q (expected Key: Value)", h)
					}
					opts.Headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
			}

			// Resolve profile
			base := cfg.Defaults
			if opts.Profile != "" {
				if resolved, err := cfg.ResolveProfile(opts.Profile); err == nil {
					base = resolved
				}
			}
			merged := curl.Merge(base, opts)

			if printCurl {
				fmt.Println(curl.ToCurl(merged))
				fmt.Println()
			}

			var clientOpts []curl.ClientOption
			clientOpts = append(clientOpts, curl.WithDefaults(cfg.Defaults))
			if logger := cfg.BuildLogger(); logger != nil {
				clientOpts = append(clientOpts, curl.WithLogger(logger))
			}

			client := curl.New(clientOpts...)
			defer client.Close()

			resp, err := client.Do(context.Background(), opts)
			if err != nil {
				return err
			}

			if output != "" {
				return os.WriteFile(output, []byte(resp.Body), 0644)
			}

			if jsonOutput {
				data, err := resp.JSON()
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			fmt.Println(resp.Format())
			return nil
		},
	}

	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Headers (repeatable: -H \"Key: Value\")")
	cmd.Flags().StringVarP(&data, "data", "d", "", "Request body")
	cmd.Flags().StringVar(&ja3, "ja3", "", "JA3 fingerprint")
	cmd.Flags().StringVar(&ja4r, "ja4r", "", "JA4 raw fingerprint")
	cmd.Flags().StringVar(&h2fp, "h2fp", "", "HTTP/2 fingerprint")
	cmd.Flags().StringVar(&userAgent, "user-agent", "", "User-Agent string")
	cmd.Flags().StringVar(&proxy, "proxy", "", "Proxy URL")
	cmd.Flags().IntVar(&timeout, "timeout", 30, "Timeout in seconds")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "Skip TLS verification")
	cmd.Flags().BoolVar(&noRedirect, "no-redirect", false, "Disable following redirects")
	cmd.Flags().BoolVar(&http1, "http1", false, "Force HTTP/1.1")
	cmd.Flags().BoolVar(&http3, "http3", false, "Force HTTP/3")
	cmd.Flags().StringVar(&profile, "profile", "", "Use named profile from config")
	cmd.Flags().BoolVar(&printCurl, "print-curl", false, "Print equivalent curl command")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Write response body to file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output response as JSON")

	return cmd
}

// --- batch ---

func batchCmd() *cobra.Command {
	var (
		file      string
		stdin     bool
		profile   string
		printCurl bool
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Make multiple HTTP requests in parallel",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()

			var requests []curl.RequestOptions

			if stdin {
				if err := json.NewDecoder(os.Stdin).Decode(&requests); err != nil {
					return fmt.Errorf("failed to parse stdin: %w", err)
				}
			} else if file != "" {
				data, err := os.ReadFile(file)
				if err != nil {
					return fmt.Errorf("read file: %w", err)
				}
				if err := json.Unmarshal(data, &requests); err != nil {
					return fmt.Errorf("parse file: %w", err)
				}
			} else {
				return fmt.Errorf("provide --file or --stdin")
			}

			if profile != "" {
				for i := range requests {
					if requests[i].Profile == "" {
						requests[i].Profile = profile
					}
				}
			}

			if printCurl {
				for i := range requests {
					requests[i].PrintCurl = true
				}
			}

			var clientOpts []curl.ClientOption
			clientOpts = append(clientOpts, curl.WithDefaults(cfg.Defaults))
			if logger := cfg.BuildLogger(); logger != nil {
				clientOpts = append(clientOpts, curl.WithLogger(logger))
			}

			client := curl.New(clientOpts...)
			defer client.Close()

			results, err := client.Batch(context.Background(), requests)
			if err != nil {
				return err
			}

			if jsonOut {
				out, _ := json.MarshalIndent(results, "", "  ")
				fmt.Println(string(out))
				return nil
			}

			for _, r := range results {
				fmt.Printf("--- [%d] %s ---\n", r.Index, r.URL)
				if r.Error != "" {
					fmt.Printf("ERROR: %s\n", r.Error)
				} else if r.Response != nil {
					if r.CurlCmd != "" {
						fmt.Printf("Curl: %s\n\n", r.CurlCmd)
					}
					fmt.Println(r.Response.Format())
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON file with array of request objects")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read request array from stdin")
	cmd.Flags().StringVar(&profile, "profile", "", "Apply named profile to all requests")
	cmd.Flags().BoolVar(&printCurl, "print-curl", false, "Print curl commands for all requests")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON array")

	return cmd
}

// --- config ---

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	cmd.AddCommand(configInitCmd(), configPathCmd(), configShowCmd(), configValidateCmd())
	return cmd
}

func configInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create default config.jsonc",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := config.DefaultConfigPath()
			if !force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
				}
			}
			created, err := config.EnsureDefaultConfig()
			if err != nil {
				return err
			}
			fmt.Printf("Config created at %s\n", created)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config")
	return cmd
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print config file path",
		Run: func(cmd *cobra.Command, args []string) {
			if configPath != "" {
				fmt.Println(configPath)
				return
			}
			path, err := config.FindConfigFile()
			if err != nil {
				fmt.Println(config.DefaultConfigPath(), "(not found, would be created here)")
				return
			}
			fmt.Println(path)
		},
	}
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print resolved config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			out, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}
}

func configValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [path]",
		Short: "Validate a config.jsonc file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := configPath
			if len(args) > 0 {
				path = args[0]
			}
			if path == "" {
				var err error
				path, err = config.FindConfigFile()
				if err != nil {
					return fmt.Errorf("no config file found")
				}
			}
			_, err := config.LoadConfig(path)
			if err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}
			fmt.Printf("Config %s is valid\n", path)
			return nil
		},
	}
}

// --- version ---

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	}
}

// --- helpers ---

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
