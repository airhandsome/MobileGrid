package android

import (
	"github.com/google/uuid"
	"os/exec"
	"strings"
)

var KeyCodeToAndroid = map[string]int{
	"Digit0":       7,
	"Digit1":       8,
	"Digit2":       9,
	"Digit3":       10,
	"Digit4":       11,
	"Digit5":       12,
	"Digit6":       13,
	"Digit7":       14,
	"Digit8":       15,
	"Digit9":       16,
	"Numpad0":      7,
	"Numpad1":      8,
	"Numpad2":      9,
	"Numpad3":      10,
	"Numpad4":      11,
	"Numpad5":      12,
	"Numpad6":      13,
	"Numpad7":      14,
	"Numpad8":      15,
	"Numpad9":      16,
	"Star":         17,
	"Pound":        18,
	"__VolumeUp":   24,
	"__VolumeDown": 25,
	"__Power":      26,
	"KeyA":         29,
	"KeyB":         30,
	"KeyC":         31,
	"KeyD":         32,
	"KeyE":         33,
	"KeyF":         34,
	"KeyG":         35,
	"KeyH":         36,
	"KeyI":         37,
	"KeyJ":         38,
	"KeyK":         39,
	"KeyL":         40,
	"KeyM":         41,
	"KeyN":         42,
	"KeyO":         43,
	"KeyP":         44,
	"KeyQ":         45,
	"KeyR":         46,
	"KeyS":         47,
	"KeyT":         48,
	"KeyU":         49,
	"KeyV":         50,
	"KeyW":         51,
	"KeyX":         52,
	"KeyY":         53,
	"KeyZ":         54,
	"Comma":        55,
	"Period":       56,
	"AltLeft":      57,
	"AltRight":     58,
	"ShiftLeft":    59,
	"ShiftRight":   60,
	"Tab":          61,
	"Space":        62,
	"Enter":        66,
	"Delete":       67,
	"Minus":        69,
	"Equal":        70,
	"BracketLeft":  71,
	"BracketRight": 72,
	"Backslash":    73,
	"Semicolon":    74,
	"Quote":        75,
	"Slash":        76,
	"Backspace":    67,
	"ArrowUp":      19,
	"ArrowDown":    20,
	"ArrowLeft":    21,
	"ArrowRight":   22,
	"Back":         4,
	"Home":         3,
	"Menu":         82,
	"App_Switch":   187,
	"WakeUp":       224,
}

const (
	CONTROL_MSG_TYPE_INJECT_KEYCODE = iota
	CONTROL_MSG_TYPE_INJECT_TEXT
	CONTROL_MSG_TYPE_INJECT_TOUCH_EVENT
	CONTROL_MSG_TYPE_INJECT_SCROLL_EVENT
	CONTROL_MSG_TYPE_BACK_OR_SCREEN_ON
	CONTROL_MSG_TYPE_EXPAND_NOTIFICATION_PANEL
	CONTROL_MSG_TYPE_COLLAPSE_NOTIFICATION_PANEL
	CONTROL_MSG_TYPE_GET_CLIPBOARD
	CONTROL_MSG_TYPE_SET_CLIPBOARD
	CONTROL_MSG_TYPE_SET_SCREEN_POWER_MODE
	CONTROL_MSG_TYPE_ROTATE_DEVICE
	CONTROL_MIDIFY_IMAGE_SETTING
	CONTROL_MSG_TYPE_RESET_VIDEO_STREAM
	CONTROL_MSG_TYPE_RESET_IMAGE_STREAM
	CONTROL_MSG_TYPE_MINIPROGRAM
	SCREEN_ORIENTATION_LANDSCAPE
	SCREEN_ORIENTATION_PORTRAIT
	CONTROL_END_UPLOAD
	RECORD_SERVER_SPEED
	CNTROL_WAKE_UP
)

var DeviceTempPath = "/data/local/tmp"

func float2byte(x float64) byte {
	return byte(x)
}

var adbPath string

func AdbPath() string {
	if adbPath == "" {
		adbPath, _ = exec.LookPath("adb")
	}
	return adbPath
}

func IdGenerator(length int) string {
	// 生成一个UUID
	u := uuid.New()
	// 将UUID转换为字符串并去掉所有的-
	uuidStr := u.String()
	uuidStr = strings.ReplaceAll(uuidStr, "-", "")

	// 如果请求的长度大于UUID字符串的长度，则返回整个UUID字符串
	if length >= len(uuidStr) {
		return uuidStr
	}

	// 返回指定长度的子字符串
	return uuidStr[:length]
}
func contains(l []string, v string) bool {
	for _, r := range l {
		if r == v {
			return true
		}
	}
	return false
}
