package tools

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func GetMongoExecutable(installDir string, executable string) (string, error) {
	base := strings.TrimSpace(installDir)
	name := strings.TrimSpace(executable)
	if base == "" || name == "" {
		return "", fmt.Errorf("mongo install dir and executable are required")
	}
	path := filepath.Join(base, "bin", name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("mongo executable not found at %s: %w", path, err)
	}
	return path, nil
}

func BuildMongoURI(host string, port int, username string, password string, authDatabase string, connectionURI string) (string, error) {
	if uri := strings.TrimSpace(connectionURI); uri != "" {
		return uri, nil
	}
	if strings.TrimSpace(host) == "" || port <= 0 {
		return "", fmt.Errorf("mongo host and port are required")
	}

	u := &url.URL{
		Scheme: "mongodb",
		Host:   fmt.Sprintf("%s:%d", strings.TrimSpace(host), port),
		Path:   "/",
	}
	if strings.TrimSpace(username) != "" {
		if strings.TrimSpace(password) != "" {
			u.User = url.UserPassword(strings.TrimSpace(username), password)
		} else {
			u.User = url.User(strings.TrimSpace(username))
		}
	}
	query := url.Values{}
	if strings.TrimSpace(authDatabase) != "" {
		query.Set("authSource", strings.TrimSpace(authDatabase))
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func VerifyMongoInstallation(installDir string) error {
	var missing []string
	for _, executable := range []string{"mongodump", "mongorestore"} {
		path := filepath.Join(strings.TrimSpace(installDir), "bin", executable)
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, fmt.Sprintf("%s missing at %s: %v", executable, path, err))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s", strings.Join(missing, "\n"))
	}
	return nil
}

func ParseMongoPort(value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid mongo port: %w", err)
	}
	return port, nil
}
