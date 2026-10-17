//go:build windows
// +build windows

package main

import (
	"bufio"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"recol/pkg/imagepkg"
	"recol/pkg/installpkg"
)

var (
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	procChooseColorW = comdlg32.NewProc("ChooseColorW")
)

const (
	CC_RGBINIT = 0x00000001
	CC_FULLOPEN = 0x00000002
)

type CHOOSECOLOR struct {
	lStructSize    uint32
	hwndOwner      uintptr
	hInstance      uintptr
	rgbResult      uint32
	lpCustColors   *uint32
	Flags          uint32
	lCustData      uintptr
	lpfnHook       uintptr
	lpTemplateName *uint16
}

// ShowColorPicker opens the native Windows color picker dialog.
// Returns the selected color and a boolean indicating if a color was chosen.
func ShowColorPicker() (color.Color, bool) {
	var customColors [16]uint32

	cc := CHOOSECOLOR{
		lStructSize: uint32(unsafe.Sizeof(CHOOSECOLOR{})),
		Flags:       CC_RGBINIT | CC_FULLOPEN,
		lpCustColors: &customColors[0],
	}

	ret, _, _ := procChooseColorW.Call(uintptr(unsafe.Pointer(&cc)))
	if ret == 0 {
		return nil, false // User canceled
	}

	// Windows color format is 0x00BBGGRR
	r := uint8(cc.rgbResult & 0xFF)
	g := uint8((cc.rgbResult >> 8) & 0xFF)
	b := uint8((cc.rgbResult >> 16) & 0xFF)

	return color.NRGBA{R: r, G: g, B: b, A: 255}, true
}

func handleInteractiveMenu() {
	fmt.Println("=================================")
	fmt.Println("          recol Setup            ")
	fmt.Println("=================================")
	fmt.Println("Please run this executable as Administrator to Install/Uninstall.")
	fmt.Println()
	fmt.Println("1. Install (Add to Context Menu)")
	fmt.Println("2. Uninstall (Remove from Context Menu)")
	fmt.Println("3. Exit")
	fmt.Print("\nEnter choice (1/2/3): ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		fmt.Println("Installing...")
		if err := installpkg.Install(); err != nil {
			fmt.Printf("Error during installation: %v\n", err)
		}
	case "2":
		fmt.Println("Uninstalling...")
		if err := installpkg.Uninstall(); err != nil {
			fmt.Printf("Error during uninstallation: %v\n", err)
		}
	case "3":
		fmt.Println("Exiting.")
	default:
		fmt.Println("Invalid choice.")
	}

	fmt.Println("Press Enter to exit...")
	reader.ReadString('\n')
}

func main() {
	if len(os.Args) < 2 {
		handleInteractiveMenu()
		return
	}

	targetPath := os.Args[1]

	// Show color picker
	targetColor, ok := ShowColorPicker()
	if !ok {
		fmt.Println("Canceled.")
		return
	}

	// Process path
	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if fileInfo.IsDir() {
		err = filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && imagepkg.IsSupported(path) {
				fmt.Printf("Processing %s...\n", path)
				if err := imagepkg.ProcessImage(path, targetColor); err != nil {
					fmt.Printf("Failed to process %s: %v\n", path, err)
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error walking directory: %v\n", err)
		}
	} else {
		if imagepkg.IsSupported(targetPath) {
			fmt.Printf("Processing %s...\n", targetPath)
			if err := imagepkg.ProcessImage(targetPath, targetColor); err != nil {
				fmt.Printf("Failed to process %s: %v\n", targetPath, err)
			}
		} else {
			fmt.Println("Unsupported file type.")
		}
	}

	fmt.Println("Done.")
}
