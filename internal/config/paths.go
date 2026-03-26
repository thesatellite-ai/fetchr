package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	appName    = "fetchr"
	configFile = "config.jsonc"
	localFile  = "fetchr.jsonc"
)

// FindConfigFile searches platform-specific paths and returns the first found config file.
func FindConfigFile() (string, error) {
	for _, p := range configSearchPaths() {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}

// DefaultConfigPath returns the platform default config file path.
func DefaultConfigPath() string {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", appName, configFile)
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, appName, configFile)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", appName, configFile)
	default: // linux and others
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, appName, configFile)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", appName, configFile)
	}
}

func configSearchPaths() []string {
	home, _ := os.UserHomeDir()
	paths := []string{localFile}

	switch runtime.GOOS {
	case "darwin":
		paths = append(paths,
			filepath.Join(home, ".config", appName, configFile),
			filepath.Join(home, "Library", "Application Support", appName, configFile),
		)
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			paths = append(paths, filepath.Join(appData, appName, configFile))
		}
		paths = append(paths, filepath.Join(home, ".config", appName, configFile))
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			paths = append(paths, filepath.Join(xdg, appName, configFile))
		}
		paths = append(paths, filepath.Join(home, ".config", appName, configFile))
	}

	return paths
}
