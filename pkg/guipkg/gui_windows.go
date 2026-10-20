//go:build windows
// +build windows

package guipkg

import (
	"image/color"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")

	procMessageBoxW = user32.NewProc("MessageBoxW")
	procChooseColorW = comdlg32.NewProc("ChooseColorW")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

const (
	MB_YESNO = 0x00000004
	MB_ICONQUESTION = 0x00000020
	IDYES = 6

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

func getAsyncKeyState(vKey int) uint16 {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vKey))
	return uint16(ret)
}

// ShowMainDialog opens a custom GUI using lxn/walk which looks native but allows
// for more complex layouts like a color preview and mode selection.
func ShowMainDialog() (color.Color, string, bool) {
	var mw *walk.MainWindow
	var colorPreview *walk.CustomWidget
	var modeCombo *walk.ComboBox
	var pickedColor color.Color = nil
	var accepted bool = false

	MainWindow{
		AssignTo: &mw,
		Title:    "Recolor",
		MinSize:  Size{Width: 350, Height: 250},
		Size:     Size{Width: 350, Height: 250},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "Выбранный цвет:"},
					CustomWidget{
						AssignTo: &colorPreview, // We'll re-assign this to a *walk.CustomWidget
						MinSize: Size{Width: 50, Height: 20},
						Paint: func(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
							if pickedColor == nil {
								return nil
							}
							r, g, b, _ := pickedColor.RGBA()
							brush, err := walk.NewSolidColorBrush(walk.RGB(byte(r>>8), byte(g>>8), byte(b>>8)))
							if err != nil {
								return err
							}
							defer brush.Dispose()
							return canvas.FillRectangle(brush, updateBounds)
						},
					},
				},
			},
			PushButton{
				Text: "Выбрать цвет из палитры",
				OnClicked: func() {
					c, ok := ShowColorPicker()
					if ok {
						pickedColor = c
						colorPreview.Invalidate()
					}
				},
			},
			PushButton{
				Text: "Выбрать цвет пипеткой (с экрана)",
				OnClicked: func() {
					c, ok := PickColorFromScreen(mw)
					if ok {
						pickedColor = c
						colorPreview.Invalidate()
					}
				},
			},
			Label{Text: "Режим наложения:"},
			ComboBox{
				AssignTo: &modeCombo,
				Model:    []string{"Обычный (HSL)", "Умножение (с перекраской белого)"},
				CurrentIndex: 0,
			},
			VSpacer{},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "Применить",
						OnClicked: func() {
							if pickedColor != nil {
								accepted = true
								mw.Close()
							} else {
								walk.MsgBox(mw, "Ошибка", "Пожалуйста, выберите цвет", walk.MsgBoxIconWarning)
							}
						},
					},
					PushButton{
						Text: "Отмена",
						OnClicked: func() {
							mw.Close()
						},
					},
				},
			},
		},
	}.Create()

	mw.Run()

	if !accepted || pickedColor == nil {
		return nil, "", false
	}

	mode := "normal"
	if modeCombo.CurrentIndex() == 1 {
		mode = "multiply"
	}

	return pickedColor, mode, true
}

// PickColorFromScreen enters a screen picking mode.
func PickColorFromScreen(owner *walk.MainWindow) (color.Color, bool) {
	// A quick hack for a color picker:
	// We minimize the main window, wait a bit, then grab the cursor position and pixel color on left click.
	// Implementing a full transparent overlay in WinAPI is complex.
	// Instead, we will use a loop polling for GetAsyncKeyState.

	walk.MsgBox(owner, "Пипетка", "Наведите мышь на нужный пиксель на экране и нажмите ПРОБЕЛ.", walk.MsgBoxIconInformation)

	for {
		// VK_SPACE is 0x20
		if getAsyncKeyState(0x20)&0x8000 != 0 {
			var pt win.POINT
			win.GetCursorPos(&pt)

			hdc := win.GetDC(0)
			if hdc == 0 {
				return nil, false
			}

			c := win.GetPixel(hdc, pt.X, pt.Y)
			win.ReleaseDC(0, hdc)

			r := uint8(c & 0xFF)
			g := uint8((c >> 8) & 0xFF)
			b := uint8((c >> 16) & 0xFF)

			return color.NRGBA{R: r, G: g, B: b, A: 255}, true
		}

		// Also allow escape to cancel (VK_ESCAPE is 0x1B)
		if getAsyncKeyState(0x1B)&0x8000 != 0 {
			return nil, false
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// Ensure the old functions are still here for compatibility if needed.
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

func ShowInstallerDialog() (string, error) {
	textPtr, _ := syscall.UTF16PtrFromString("Программа recol.\nДа - Установить в контекстное меню\nНет - Удалить\nОтмена - Выход")
	captionPtr, _ := syscall.UTF16PtrFromString("Установка / Удаление")
	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(3|MB_ICONQUESTION),
	)
	if ret == 6 {
		return "install", nil
	} else if ret == 7 {
		return "uninstall", nil
	}
	return "cancel", nil
}

func ShowMessage(title, text string) {
	textPtr, _ := syscall.UTF16PtrFromString(text)
	captionPtr, _ := syscall.UTF16PtrFromString(title)
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(captionPtr)), 0)
}
