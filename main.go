package main

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>
#include <stdlib.h>
#include <locale.h>
#include <string.h>
#include <X11/cursorfont.h>

XIC xic;
XIM xim;

void initializeXIM(Display *display, Window window) {
    setlocale(LC_ALL, "");
    XSetLocaleModifiers("");
    xim = XOpenIM(display, NULL, NULL, NULL);
    if (!xim) {
        XSetLocaleModifiers("@im=none");
        xim = XOpenIM(display, NULL, NULL, NULL);
    }
    if (xim) {
        xic = XCreateIC(xim,
                        XNInputStyle, XIMPreeditNothing | XIMStatusNothing,
                        XNClientWindow, window,
                        XNFocusWindow, window,
                        NULL);
    }
}

*/
import "C"
import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"time"
	"unsafe"
)

func generateImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	green := color.RGBA{0, 255, 0, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{green}, image.Point{}, draw.Src)
	return img
}

func drawImage(display *C.Display, window C.Window, gc C.GC, img *image.RGBA) {
	if len(img.Pix) < 1 {
		return
	}

	width := C.uint(img.Rect.Dx())
	height := C.uint(img.Rect.Dy())
	data := (*C.char)(unsafe.Pointer(&img.Pix[0]))

	screen := C.XDefaultScreen(display)
	visual := C.XDefaultVisual(display, screen)

	ximage := C.XCreateImage(
		display,
		visual,
		24,        // depth
		C.ZPixmap, // format
		0,         // offset
		data,      // data
		width,
		height,
		32, // bitmap_pad
		0,  // bytes_per_line
	)

	C.XPutImage(display, window, gc, ximage, 0, 0, 0, 0, width, height)

	// Освобождение памяти ximage вручную
	C.free(unsafe.Pointer(ximage))
}

func changeWindowTitle(display *C.Display, window C.Window, title string) {
	ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(ctitle))
	C.XStoreName(display, window, ctitle)
	C.XFlush(display)
}

func handleConfigureNotify(event *C.XConfigureEvent, display *C.Display, window C.Window, gc C.GC) {
	width := int(event.width)
	height := int(event.height)
	fmt.Printf("Window resized to %dx%d\n", width, height)

	// Generate image of the new size
	img := generateImage(width, height)

	// Draw the image
	drawImage(display, window, gc, img)
}

func call_go_function_key_press(keycode C.int) {
	fmt.Printf("Key pressed: %d\n", keycode)
}

func call_go_function_button_press(button C.int, x C.int, y C.int) {
	fmt.Printf("Button pressed: %d at (%d, %d)\n", button, x, y)
}

func call_go_function_pointer_motion(x C.int, y C.int) {
	//fmt.Printf("Pointer moved to (%d, %d)\n", x, y)
}

func resizeWindow(display *C.Display, window C.Window, width, height int) {
	C.XResizeWindow(display, window, C.uint(width), C.uint(height))
	C.XFlush(display)
}

func changeCursor(display *C.Display, window C.Window, cursorShape uint) {
	cursor := C.XCreateFontCursor(display, C.uint(cursorShape))
	C.XDefineCursor(display, window, cursor)
	C.XFlush(display)
}

func handleKeyPress(event *C.XKeyEvent, display *C.Display) {
	var buffer [32]C.char
	var keysym C.KeySym
	var status C.Status

	// Преобразование события нажатия клавиши в строку с учетом локали
	keycount := C.Xutf8LookupString(C.xic, event, &buffer[0], C.int(len(buffer)), &keysym, &status)
	if keycount > 0 {
		keyStr := C.GoStringN(&buffer[0], C.int(keycount))
		fmt.Printf("Key pressed: %d, Char: %s\n", event.keycode, keyStr)
	} else {
		fmt.Printf("Key pressed: %d\n", event.keycode)
	}

	// Определение состояния модификаторов
	shiftPressed := event.state&C.ShiftMask != 0
	controlPressed := event.state&C.ControlMask != 0
	altPressed := event.state&C.Mod1Mask != 0
	capsLockPressed := event.state&C.LockMask != 0

	fmt.Printf("Ctrl: %v, Alt: %v, Shift: %v, CapsLock: %v\n",
		controlPressed, altPressed, shiftPressed, capsLockPressed)
}

func printUserInput(display *C.Display, event *C.XKeyEvent) {
	// Translate the state of the keyboard buffer to a human-readable string
	var buf [32]C.char
	keycount := C.XLookupString(event, &buf[0], 32, nil, nil)
	if keycount > 0 {
		keyStr := C.GoStringN(&buf[0], C.int(keycount))
		fmt.Printf("User input: %s\n", keyStr)
	}
}

func setWindowSizeHints(display *C.Display, window C.Window, minWidth, minHeight, maxWidth, maxHeight int) {
	var hints C.XSizeHints
	hints.flags = C.PMinSize | C.PMaxSize
	hints.min_width = C.int(minWidth)
	hints.min_height = C.int(minHeight)
	hints.max_width = C.int(maxWidth)
	hints.max_height = C.int(maxHeight)
	C.XSetWMNormalHints(display, window, &hints)
	C.XFlush(display)
}

func moveWindow(display *C.Display, window C.Window, x, y int) {
	C.XMoveWindow(display, window, C.int(x), C.int(y))
	C.XFlush(display)
}

func closeWindow(display *C.Display, window C.Window) {
	C.XDestroyWindow(display, window)
	C.XCloseDisplay(display)
	os.Exit(0)
}

var counterClose = 0

// Функция для предотвращения закрытия окна
func preventClose(display *C.Display, window C.Window) {
	fmt.Println("Attempting to close window. Perform necessary actions before closing.")
	// Здесь можно вставить код для сохранения данных или отображения диалогового окна
	counterClose++
	if counterClose > 3 {
		closeWindow(display, window)
	}
}

func handleVisibilityNotify(event *C.XVisibilityEvent) {
	switch event.state {
	case C.VisibilityUnobscured:
		fmt.Println("Window is fully visible")
	case C.VisibilityPartiallyObscured:
		fmt.Println("Window is partially obscured")
	case C.VisibilityFullyObscured:
		fmt.Println("Window is fully obscured")
	}
}

func handleEnterNotify(event *C.XCrossingEvent) {
	fmt.Println("Pointer entered the window")
}

func handleLeaveNotify(event *C.XCrossingEvent) {
	fmt.Println("Pointer left the window")
}

func main() {
	display := C.XOpenDisplay(nil)
	if display == nil {
		fmt.Println("Unable to open X display")
		return
	}
	defer C.XCloseDisplay(display)

	screen := C.XDefaultScreen(display)
	root := C.XRootWindow(display, screen)
	window := C.XCreateSimpleWindow(display, root, 10, 10, 400, 300, 1, C.XBlackPixel(display, screen), C.XWhitePixel(display, screen))

	msg := C.CString("Hello, World!")
	defer C.free(unsafe.Pointer(msg))
	C.XSetStandardProperties(display, window, msg, msg, 0, nil, 0, nil)
	C.XSelectInput(display, window, C.ExposureMask|C.StructureNotifyMask|C.KeyPressMask|C.ButtonPressMask|C.PointerMotionMask|C.StructureNotifyMask|C.FocusChangeMask|C.VisibilityChangeMask|C.EnterWindowMask|C.LeaveWindowMask)

	C.initializeXIM(display, window)

	wmDeleteMessage := C.XInternAtom(display, C.CString("WM_DELETE_WINDOW"), C.False)
	C.XSetWMProtocols(display, window, &wmDeleteMessage, 1)
	C.XMapWindow(display, window)

	gc := C.XCreateGC(display, C.Drawable(window), 0, nil)
	defer C.XFreeGC(display, gc)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var event C.XEvent
	for {
		select {
		case <-ticker.C:
			// Обновление содержимого окна каждую секунду
			xconfigure := (*C.XConfigureEvent)(unsafe.Pointer(&event))
			handleConfigureNotify(xconfigure, display, window, gc)
		default:
			if C.XPending(display) > 0 {
				C.XNextEvent(display, &event)
				eventType := *(*C.int)(unsafe.Pointer(&event))
				switch eventType {
				case C.Expose:
					// Initial draw when the window is exposed
					xconfigure := (*C.XConfigureEvent)(unsafe.Pointer(&event))
					handleConfigureNotify(xconfigure, display, window, gc)
				case C.KeyPress:
					xkey := (*C.XKeyEvent)(unsafe.Pointer(&event))
					handleKeyPress(xkey, display)
				case C.ButtonPress:
					xbutton := (*C.XButtonEvent)(unsafe.Pointer(&event))
					call_go_function_button_press(C.int(xbutton.button), C.int(xbutton.x), C.int(xbutton.y))
					//changeWindowTitle(display, window, "asdasdas")
					//resizeWindow(display, window, 500, 100)
					//changeCursor(display, window, C.XC_hand2)
					//setWindowSizeHints(display, window, 100, 100, 300, 300)
					moveWindow(display, window, 100, 100)
				case C.MotionNotify:
					xmotion := (*C.XMotionEvent)(unsafe.Pointer(&event))
					call_go_function_pointer_motion(C.int(xmotion.x), C.int(xmotion.y))
				case C.ConfigureNotify:
					xconfigure := (*C.XConfigureEvent)(unsafe.Pointer(&event))
					handleConfigureNotify(xconfigure, display, window, gc)
				case C.ClientMessage:
					xclient := (*C.XClientMessageEvent)(unsafe.Pointer(&event))
					if C.Atom(*(*C.long)(unsafe.Pointer(&xclient.data))) == wmDeleteMessage {
						preventClose(display, window)
					}
				case C.VisibilityNotify:
					xvisibility := (*C.XVisibilityEvent)(unsafe.Pointer(&event))
					handleVisibilityNotify(xvisibility)
				case C.UnmapNotify:
					fmt.Println("Window minimized")
				case C.MapNotify:
					fmt.Println("Window restored")
				case C.FocusIn:
					fmt.Println("Window gained focus")
				case C.FocusOut:
					fmt.Println("Window lost focus")
				case C.EnterNotify:
					xcrossing := (*C.XCrossingEvent)(unsafe.Pointer(&event))
					handleEnterNotify(xcrossing)
				case C.LeaveNotify:
					xcrossing := (*C.XCrossingEvent)(unsafe.Pointer(&event))
					handleLeaveNotify(xcrossing)
				}
			}
		}
	}

}
