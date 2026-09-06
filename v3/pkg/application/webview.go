package application

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/wailsapp/wails/v3/internal/assetserver"
)

// A webview added to a window, beside the webview the window was created with.
//
// A window creates one webview and resizes it to fill the window. An application
// that composites its own views over a page needs more than one webview in the
// same window: a terminal, a browser or a preview, each a separate document,
// each placed by the application, and each drawn by the system rather than by
// the page, which cannot draw over them.
//
// An added webview is created from the window's configuration, so it loads from
// the same scheme, calls the same bindings and receives the same events. That
// also means it carries the window's authority: a page loaded into it can call
// every binding the window's own page can call. Load only content the
// application controls, or content whose origin it trusts.
//
// The caller sets its position and size, and it does not resize with the window.
//
// Add one after the window's own page has loaded. An added webview runs the same
// runtime, and that runtime reports itself ready through the window, so one added
// before the window's page has loaded makes the window report a runtime that page
// cannot run yet: JavaScript held for it is sent and dropped.

// ErrWebviewUnsupported is returned by AddWebview where the platform has no
// implementation.
var ErrWebviewUnsupported = errors.New("adding a webview to a window is not implemented on this platform")

// ErrWebviewWindowGone is returned by AddWebview for a window that is closed or
// was never created.
var ErrWebviewWindowGone = errors.New("the window cannot take a webview")

// ErrWebviewNotTransparent is returned by AddWebview when Transparent was asked
// for and the platform no longer turns the webview's own background off.
var ErrWebviewNotTransparent = errors.New("the webview's background cannot be turned off")

// WebviewOptions is the configuration for a webview added to a window.
type WebviewOptions struct {
	// URL the webview loads. A relative URL is served by the application, as a
	// window's URL is.
	URL string

	// Position and size inside the window's content, in points from its top
	// left.
	//
	// Fractional points are accepted. The frame is aligned inward to the
	// display's pixels, so each edge lands on a pixel and the frame does not
	// grow past the requested rect.
	X, Y, Width, Height float64

	// The colour the webview displays before its document renders. Applied on
	// macOS 12 and later.
	BackgroundColour RGBA

	// Whether the webview renders its own background. A webview renders only the
	// area it has laid out; with a background it fills the rest with opaque
	// white, and without one that area is clear.
	Transparent bool

	// Whether the webview is hidden. A hidden webview keeps its document.
	Hidden bool
}

// Webview is a webview added to a window.
type Webview struct {
	window *WebviewWindow

	// The platform view, read and cleared on the main thread only. Every call
	// reaches it there, so a call and a close cannot overlap and no lock is
	// needed: a lock held across a call to the main thread would be taken by the
	// main thread itself when a window is destroyed.
	impl webviewImpl
}

// webviewImpl is the platform implementation.
type webviewImpl interface {
	setBounds(x, y, width, height float64)
	setHidden(hidden bool)
	setURL(url string)
	execJS(js string)
	nativeView() unsafe.Pointer
	destroy()
}

// The webviews added to each window. Kept here rather than on WebviewWindow so
// that the window carries no field on platforms without an implementation.
var (
	addedMutex sync.Mutex
	added      = map[*WebviewWindow][]*Webview{}
)

// webviews returns the webviews added to this window.
func (w *WebviewWindow) webviews() []*Webview {
	addedMutex.Lock()
	defer addedMutex.Unlock()
	return append([]*Webview(nil), added[w]...)
}

// dropWebviews closes every webview added to this window and forgets them. The
// window's own teardown calls it. The content view holds the views, and this
// releases what this package holds.
func (w *WebviewWindow) dropWebviews() {
	addedMutex.Lock()
	held := added[w]
	delete(added, w)
	addedMutex.Unlock()
	for _, view := range held {
		view.release()
	}
}

// AddWebview adds a webview to this window and returns it.
func (w *WebviewWindow) AddWebview(options WebviewOptions) (*Webview, error) {
	if w == nil || w.impl == nil || w.isDestroyed() {
		return nil, ErrWebviewWindowGone
	}
	// A relative URL is served by the application, as a window's URL is.
	url, err := assetserver.GetStartURL(options.URL)
	if err != nil {
		return nil, err
	}
	options.URL = url

	var held *Webview
	var impl webviewImpl
	var made error
	InvokeSync(func() {
		// Checked here rather than before: the window is destroyed on this
		// thread, so a check made anywhere else can be out of date by the time
		// the view is created from the window's configuration.
		if w.isDestroyed() {
			made = ErrWebviewWindowGone
			return
		}
		impl, made = addWebview(w, options)
		if made != nil {
			return
		}
		view := &Webview{window: w, impl: impl}
		addedMutex.Lock()
		added[w] = append(added[w], view)
		addedMutex.Unlock()
		held = view
	})
	if made != nil {
		return nil, made
	}
	return held, nil
}

// with runs fn against the platform implementation on the main thread, and does
// nothing once the webview is closed.
//
// The view is read on the main thread, which is also where it is destroyed, so
// the two are ordered without a lock. A lock held across this call would be
// taken by the main thread itself when the window is destroyed, and the two
// would wait for each other.
func (v *Webview) with(fn func(webviewImpl)) {
	if v == nil {
		return
	}
	InvokeSync(func() {
		if v.impl != nil {
			fn(v.impl)
		}
	})
}

// SetBounds sets the position and size inside the window's content, in points
// from its top left. The frame is aligned inward to the display's pixels.
func (v *Webview) SetBounds(x, y, width, height float64) {
	v.with(func(impl webviewImpl) { impl.setBounds(x, y, width, height) })
}

// SetHidden hides or shows the webview. A hidden webview keeps its document.
func (v *Webview) SetHidden(hidden bool) {
	v.with(func(impl webviewImpl) { impl.setHidden(hidden) })
}

// SetURL loads another address in the webview.
func (v *Webview) SetURL(url string) error {
	target, err := assetserver.GetStartURL(url)
	if err != nil {
		return err
	}
	v.with(func(impl webviewImpl) { impl.setURL(target) })
	return nil
}

// ExecJS runs JavaScript in the webview.
func (v *Webview) ExecJS(js string) {
	v.with(func(impl webviewImpl) { impl.execJS(js) })
}

// NativeView returns the platform view: an NSView on macOS, an HWND on Windows,
// a GtkWidget on Linux. Nil once the webview is closed, or where there is none.
func (v *Webview) NativeView() unsafe.Pointer {
	var handle unsafe.Pointer
	v.with(func(impl webviewImpl) { handle = impl.nativeView() })
	return handle
}

// Close removes the webview from its window. Calling it again does nothing.
func (v *Webview) Close() {
	if v == nil {
		return
	}
	addedMutex.Lock()
	rest := added[v.window][:0]
	for _, other := range added[v.window] {
		if other != v {
			rest = append(rest, other)
		}
	}
	if len(rest) == 0 {
		delete(added, v.window)
	} else {
		added[v.window] = rest
	}
	addedMutex.Unlock()
	v.release()
}

// release destroys the platform view once and forgets it.
func (v *Webview) release() {
	if v == nil {
		return
	}
	InvokeSync(func() {
		impl := v.impl
		v.impl = nil
		if impl != nil {
			impl.destroy()
		}
	})
}
