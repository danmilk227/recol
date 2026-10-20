//go:build !windows
// +build !windows

package guipkg

import (
	"fmt"
	"image/color"
)

func ShowMainDialog() (c color.Color, modeResult string, ok bool) {
	return nil, "", false
}

func ShowInstallerDialog() (string, error) {
	return "", fmt.Errorf("GUI not implemented on non-windows platform")
}

func ShowMessage(title, text string) {
	fmt.Printf("[%s] %s\n", title, text)
}
