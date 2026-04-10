//go:build windows

package daemon

import "fmt"

// Install is not implemented on Windows yet.
func Install(composeFile, projectName string) error {
	return fmt.Errorf("daemon install is not supported on windows yet")
}

// Uninstall is not implemented on Windows yet.
func Uninstall() error {
	return fmt.Errorf("daemon uninstall is not supported on windows yet")
}
