//go:build windows
// +build windows

package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"recol/pkg/guipkg"
	"recol/pkg/imagepkg"
	"recol/pkg/installpkg"
)

func handleInteractiveMenu() {
	action, _ := guipkg.ShowInstallerDialog()
	switch action {
	case "install":
		if err := installpkg.Install(); err != nil {
			guipkg.ShowMessage("Ошибка установки", fmt.Sprintf("Ошибка: %v\nПопробуйте запустить от имени администратора.", err))
		} else {
			guipkg.ShowMessage("Установка", "Установка прошла успешно. Теперь вы можете использовать контекстное меню.")
		}
	case "uninstall":
		if err := installpkg.Uninstall(); err != nil {
			guipkg.ShowMessage("Ошибка удаления", fmt.Sprintf("Ошибка: %v\nПопробуйте запустить от имени администратора.", err))
		} else {
			guipkg.ShowMessage("Удаление", "Программа успешно удалена.")
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		handleInteractiveMenu()
		return
	}

	targetPath := os.Args[1]

	// Multiple file selection workaround via local TCP socket
	const port = "127.0.0.1:49193" // arbitrary unused port for IPC

	var listener net.Listener
	var err error

	// Try to listen or connect in a loop to handle the race condition of simultaneous launches
	for i := 0; i < 5; i++ {
		conn, errDial := net.DialTimeout("tcp", port, 50*time.Millisecond)
		if errDial == nil {
			// Server already running, send the path and exit
			fmt.Fprintf(conn, "%s\n", targetPath)
			conn.Close()
			return
		}

		listener, err = net.Listen("tcp", port)
		if err == nil {
			break // Successfully became the server
		}

		// If both Dial and Listen failed, another instance is probably in the middle of binding.
		// Wait a tiny bit and retry.
		time.Sleep(20 * time.Millisecond)
	}

	if err != nil {
		guipkg.ShowMessage("Ошибка IPC", fmt.Sprintf("Не удалось запустить процесс: %v", err))
		return
	}
	defer listener.Close()

	paths := []string{targetPath}

	// Accept connections for a short window (e.g. 500ms) to collect paths from concurrent instances
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			reader := bufio.NewReader(conn)
			path, _ := reader.ReadString('\n')
			path = strings.TrimSpace(path)
			if path != "" {
				paths = append(paths, path)
			}
			conn.Close()
		}
	}()

	// Wait a moment to ensure all instances of the context menu launch have had time to send paths
	time.Sleep(500 * time.Millisecond)

	// Stop listening for new ones
	listener.Close()

	// Ask user for color/mode once for all collected files
	targetColor, mode, ok := guipkg.ShowMainDialog()
	if !ok {
		return
	}

	var errs []string

	for _, path := range paths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Ошибка: %v", err))
			continue
		}

		if fileInfo.IsDir() {
			err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && imagepkg.IsSupported(p) {
					if err := imagepkg.ProcessImage(p, targetColor, mode); err != nil {
						errs = append(errs, fmt.Sprintf("Не удалось обработать %s: %v", p, err))
					}
				}
				return nil
			})
			if err != nil {
				errs = append(errs, fmt.Sprintf("Ошибка чтения папки: %v", err))
			}
		} else {
			if imagepkg.IsSupported(path) {
				if err := imagepkg.ProcessImage(path, targetColor, mode); err != nil {
					errs = append(errs, fmt.Sprintf("Не удалось обработать %s: %v", path, err))
				}
			} else {
				errs = append(errs, fmt.Sprintf("Неподдерживаемый тип файла: %s", path))
			}
		}
	}

	if len(errs) > 0 {
		guipkg.ShowMessage("Завершено с ошибками", strings.Join(errs, "\n"))
	}
}
