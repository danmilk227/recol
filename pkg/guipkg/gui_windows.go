//go:build windows
// +build windows

package guipkg

import (
	"image/color"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	procMessageBoxW    = user32.NewProc("MessageBoxW")
	procChooseColorW   = comdlg32.NewProc("ChooseColorW")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	procGetCursorPos   = user32.NewProc("GetCursorPos")
	procGetDC          = user32.NewProc("GetDC")
	procReleaseDC      = user32.NewProc("ReleaseDC")
	procGetPixel       = gdi32.NewProc("GetPixel")
)

const (
	MB_OK           = 0x00000000
	MB_YESNO        = 0x00000004
	MB_ICONQUESTION = 0x00000020
	MB_ICONWARNING  = 0x00000030
	MB_YESNOCANCEL  = 0x00000003
	IDYES           = 6
	IDNO            = 7

	CC_RGBINIT  = 0x00000001
	CC_FULLOPEN = 0x00000002
)

type POINT struct {
	X int32
	Y int32
}

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

func getAsyncKeyState(vKey int) uint16 {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vKey))
	return uint16(ret)
}

// ShowMessage displays a simple native message box.
func ShowMessage(title, text string) {
	textPtr, _ := syscall.UTF16PtrFromString(text)
	captionPtr, _ := syscall.UTF16PtrFromString(title)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(captionPtr)), MB_OK)
}

// ShowInstallerDialog asks the user whether to install or uninstall.
func ShowInstallerDialog() (string, error) {
	textPtr, _ := syscall.UTF16PtrFromString("Программа recol.\nДа - Установить в контекстное меню\nНет - Удалить\nОтмена - Выход")
	captionPtr, _ := syscall.UTF16PtrFromString("Установка / Удаление recol")
	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(MB_YESNOCANCEL|MB_ICONQUESTION),
	)
	if ret == IDYES {
		return "install", nil
	} else if ret == IDNO {
		return "uninstall", nil
	}
	return "cancel", nil
}

// askYesNo presents a yes/no dialog and returns true if YES.
func askYesNo(title, text string) bool {
	textPtr, _ := syscall.UTF16PtrFromString(text)
	captionPtr, _ := syscall.UTF16PtrFromString(title)
	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(MB_YESNO|MB_ICONQUESTION),
	)
	return ret == IDYES
}

// ShowColorPicker opens the native Windows color picker dialog.
func ShowColorPicker() (color.Color, bool) {
	var customColors [16]uint32
	cc := CHOOSECOLOR{
		lStructSize: uint32(unsafe.Sizeof(CHOOSECOLOR{})),
		Flags:       CC_RGBINIT | CC_FULLOPEN,
		lpCustColors: &customColors[0],
	}

	ret, _, _ := procChooseColorW.Call(uintptr(unsafe.Pointer(&cc)))
	if ret == 0 {
		return nil, false
	}

	r := uint8(cc.rgbResult & 0xFF)
	g := uint8((cc.rgbResult >> 8) & 0xFF)
	b := uint8((cc.rgbResult >> 16) & 0xFF)

	return color.NRGBA{R: r, G: g, B: b, A: 255}, true
}

// PickColorFromScreen waits for the user to press SPACE to pick a color from the screen.
func PickColorFromScreen() (color.Color, bool) {
	ShowMessage("Пипетка", "Наведите мышь на нужный пиксель на экране и нажмите ПРОБЕЛ.\nДля отмены нажмите ESC.")

	for {
		// VK_SPACE
		if getAsyncKeyState(0x20)&0x8000 != 0 {
			var pt POINT
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

			hdc, _, _ := procGetDC.Call(0)
			if hdc == 0 {
				return nil, false
			}

			c, _, _ := procGetPixel.Call(hdc, uintptr(pt.X), uintptr(pt.Y))
			procReleaseDC.Call(0, hdc)

			r := uint8(c & 0xFF)
			g := uint8((c >> 8) & 0xFF)
			b := uint8((c >> 16) & 0xFF)

			return color.NRGBA{R: r, G: g, B: b, A: 255}, true
		}

		// VK_ESCAPE
		if getAsyncKeyState(0x1B)&0x8000 != 0 {
			return nil, false
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// ShowMainDialog sequences the configuration through standard WinAPI dialogs.
func ShowMainDialog() (color.Color, string, bool) {
	var targetColor color.Color
	var ok bool

	usePipette := askYesNo("Выбор цвета", "Использовать пипетку с экрана?\n(Нажмите Нет, чтобы открыть стандартную палитру)")

	if usePipette {
		targetColor, ok = PickColorFromScreen()
	} else {
		targetColor, ok = ShowColorPicker()
	}

	if !ok || targetColor == nil {
		return nil, "", false
	}

	mode := "normal"
	useMultiply := askYesNo("Режим наложения", "Использовать режим 'Умножение'?\n(Да - перекрашивать белый цвет)\n(Нет - Обычный HSL режим)")
	if useMultiply {
		mode = "multiply"
	}

	return targetColor, mode, true
}
