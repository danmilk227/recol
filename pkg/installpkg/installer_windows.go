//go:build windows
// +build windows

package installpkg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	appDirName  = "recol"
	appFileName = "recol.exe"
)

// Install copies the executable to Program Files and sets up the registry.
func Install() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		return fmt.Errorf("ProgramFiles environment variable not found")
	}

	targetDir := filepath.Join(programFiles, appDirName)
	targetExe := filepath.Join(targetDir, appFileName)

	// Create directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory (run as administrator?): %w", err)
	}

	// Copy executable
	if err := copyFile(exePath, targetExe); err != nil {
		return fmt.Errorf("failed to copy executable: %w", err)
	}

	// Setup Registry
	if err := setupRegistry(targetExe); err != nil {
		return fmt.Errorf("failed to setup registry: %w", err)
	}

	fmt.Println("Installation successful! You can now use the context menu.")
	return nil
}

// Uninstall removes the registry keys and the application directory.
func Uninstall() error {
	// Remove Registry entries
	removeRegistry()

	programFiles := os.Getenv("ProgramFiles")
	if programFiles != "" {
		targetDir := filepath.Join(programFiles, appDirName)
		if err := os.RemoveAll(targetDir); err != nil {
			fmt.Printf("Warning: Failed to remove directory %s: %v\n", targetDir, err)
		}
	}

	fmt.Println("Uninstallation successful!")
	return nil
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	// Rename existing target out of the way before copying,
	// because if we are currently running from there, os.Create will get "Access Denied".
	if _, err := os.Stat(dst); err == nil {
		os.Rename(dst, dst+".old")
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)

	// Try to clean up .old if possible, but it might be locked. That's fine.
	os.Remove(dst + ".old")
	return err
}

func setupRegistry(exePath string) error {
	// Create common shell command
	cmdValue := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	iconValue := fmt.Sprintf("\"%s\"", exePath)

	paths := []string{
		`Directory\shell\Recolor`,
		`SystemFileAssociations\.png\shell\Recolor`,
		`SystemFileAssociations\.jpg\shell\Recolor`,
		`SystemFileAssociations\.jpeg\shell\Recolor`,
	}

	for _, p := range paths {
		k, _, err := registry.CreateKey(registry.CLASSES_ROOT, p, registry.ALL_ACCESS)
		if err != nil {
			return err
		}
		k.SetStringValue("", "Recolor (recol)")
		k.SetStringValue("Icon", iconValue)

		cmdKey, _, err := registry.CreateKey(k, "command", registry.ALL_ACCESS)
		if err != nil {
			k.Close()
			return err
		}
		cmdKey.SetStringValue("", cmdValue)

		cmdKey.Close()
		k.Close()
	}

	return nil
}

func removeRegistry() {
	paths := []string{
		`Directory\shell\Recolor`,
		`SystemFileAssociations\.png\shell\Recolor`,
		`SystemFileAssociations\.jpg\shell\Recolor`,
		`SystemFileAssociations\.jpeg\shell\Recolor`,
	}

	for _, p := range paths {
		// Attempt to delete command subkey first
		registry.DeleteKey(registry.CLASSES_ROOT, p+`\command`)
		registry.DeleteKey(registry.CLASSES_ROOT, p)
	}
}
