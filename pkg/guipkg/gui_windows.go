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

	// Create a non-blocking dialog to instruct the user
	var dlg *walk.Dialog
	Dialog{
		AssignTo: &dlg,
		Title:    "Пипетка",
		MinSize:  Size{Width: 300, Height: 100},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "Наведите мышь на нужный пиксель на экране и нажмите ПРОБЕЛ.\nДля отмены нажмите ESC."},
		},
	}.Create(owner)
	dlg.Show()

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

			dlg.Close(1)
			return color.NRGBA{R: r, G: g, B: b, A: 255}, true
		}

		// Also allow escape to cancel (VK_ESCAPE is 0x1B)
		if getAsyncKeyState(0x1B)&0x8000 != 0 {
			dlg.Close(0)
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
	var mw *walk.MainWindow
	var action string = "cancel"

	MainWindow{
		AssignTo: &mw,
		Title:    "Установка / Удаление recol",
		MinSize:  Size{Width: 300, Height: 150},
		Size:     Size{Width: 300, Height: 150},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "Добро пожаловать в recol.\nХотите установить или удалить интеграцию с контекстным меню Windows?"},
			VSpacer{},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text: "Установить",
						OnClicked: func() {
							action = "install"
							mw.Close()
						},
					},
					PushButton{
						Text: "Удалить",
						OnClicked: func() {
							action = "uninstall"
							mw.Close()
						},
					},
					PushButton{
						Text: "Отмена",
						OnClicked: func() {
							action = "cancel"
							mw.Close()
						},
					},
				},
			},
		},
	}.Create()

	mw.Run()
	return action, nil
}

func ShowMessage(title, text string) {
	// Create a temporary hidden main window to anchor the MsgBox
	// because walk.MsgBox requires an owner, or it might not show correctly in windowsgui
	var mw *walk.MainWindow
	MainWindow{
		AssignTo: &mw,
		Visible:  false,
	}.Create()

	walk.MsgBox(mw, title, text, walk.MsgBoxIconInformation)
	mw.Dispose()
}
