//go:build !windows
// +build !windows

package installpkg

import "fmt"

func Install() error {
	return fmt.Errorf("Installer is only supported on Windows")
}

func Uninstall() error {
	return fmt.Errorf("Uninstaller is only supported on Windows")
}
