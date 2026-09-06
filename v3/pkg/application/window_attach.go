package application

import (
	"errors"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/events"
)

// Placing a window over another window's content.
//
// A window that presents part of the document below it, such as a modal or a
// picker, must be positioned inside that document, move with it and stay above
// it. A window positioned by screen coordinates alone does not: it stays where
// it was placed when the parent moves, and the window manager may place another
// window between them.
//
// A separate window is required rather than an element in the page, because a
// page cannot draw over a webview the system composites, and two webviews in one
// window both set the cursor.

// Where each attached window sits, so it can be placed again when either window
// changes size. The platform keeps the offset from the parent's frame, which is
// measured from the opposite edge, so a window that is not placed again is at
// another point in the parent's content after the parent grows.
var (
	attachedMutex sync.Mutex
	attached      = map[*WebviewWindow]attachment{}
)

type attachment struct {
	parent *WebviewWindow
	x, y   float64
	off    func()
}

// ErrAttachUnsupported is returned by Attach where the platform has no
// implementation.
var ErrAttachUnsupported = errors.New("attaching a window to another is not implemented on this platform")

// ErrAttachWindowGone is returned by Attach for a window that is closed or was
// never created.
var ErrAttachWindowGone = errors.New("the window cannot be attached")

// ErrAttachToSelf is returned by Attach for a window given itself as its parent,
// or given a window already attached to it.
var ErrAttachToSelf = errors.New("a window cannot be attached to itself")

// Attach positions this window over the parent's content, x and y points from
// its top left, and makes it a child of the parent, so it moves with the parent,
// keeps its position when the parent is resized, and is drawn above it.
//
// A child window is ordered in with its parent, so attaching a hidden window
// shows it. Attach it once it is meant to be seen.
//
// Calling it again with another point moves the window. A window already
// attached to another parent is removed from that parent first.
func (w *WebviewWindow) Attach(parent *WebviewWindow, x, y float64) error {
	if w == nil || w.impl == nil || w.isDestroyed() {
		return ErrAttachWindowGone
	}
	if parent == nil || parent.impl == nil || parent.isDestroyed() {
		return ErrAttachWindowGone
	}
	// A window cannot be its own parent, and a chain of parents cannot lead back
	// to it. The platform gives no answer for either, and a parent retains its
	// child, so a cycle is a set of windows none of which is released.
	if parent == w {
		return ErrAttachToSelf
	}
	if !attachSupported {
		return ErrAttachUnsupported
	}

	attachedMutex.Lock()
	// Both windows are checked here, under the lock the teardown of either takes,
	// so the answer cannot be out of date by the time the entry is written.
	if w.isDestroyed() || parent.isDestroyed() {
		attachedMutex.Unlock()
		return ErrAttachWindowGone
	}
	for up := parent; up != nil; {
		at, ok := attached[up]
		if !ok {
			break
		}
		if at.parent == w {
			attachedMutex.Unlock()
			return ErrAttachToSelf
		}
		up = at.parent
	}
	was, held := attached[w]
	if held && was.parent != parent {
		was.off()
		held = false
	}
	if !held {
		// Either window's height is what the point is measured from, so the
		// window is placed again whenever either changes.
		offParent := parent.OnWindowEvent(events.Common.WindowDidResize, func(*WindowEvent) {
			w.place()
		})
		offOwn := w.OnWindowEvent(events.Common.WindowDidResize, func(*WindowEvent) {
			w.place()
		})
		was.off = func() {
			offParent()
			offOwn()
		}
	}
	attached[w] = attachment{parent: parent, x: x, y: y, off: was.off}
	attachedMutex.Unlock()

	w.place()
	return nil
}

// place puts the window where it was last attached.
func (w *WebviewWindow) place() {
	// The attachment is read on the main thread, which is also where it is
	// undone. Reading it first and placing afterwards would let a Detach run
	// between the two, and this would attach the window again with nothing left
	// to detach it.
	InvokeSync(func() {
		attachedMutex.Lock()
		at, held := attached[w]
		attachedMutex.Unlock()
		if !held || w.isDestroyed() || at.parent.isDestroyed() {
			return
		}
		attachWindow(w, at.parent, at.x, at.y)
	})
}

// Detach removes this window from the parent it was attached to. It keeps its
// position on screen and no longer moves with the parent.
func (w *WebviewWindow) Detach() {
	if w == nil || w.impl == nil {
		return
	}
	forget(w)
	// Always issued. The platform holds the relationship, and this package
	// forgetting it - which the parent's teardown does - does not end it.
	InvokeSync(func() {
		if w.isDestroyed() {
			return
		}
		detachWindow(w)
	})
}

// forget drops this window's attachment and returns whether it had one.
func forget(w *WebviewWindow) bool {
	attachedMutex.Lock()
	defer attachedMutex.Unlock()
	at, held := attached[w]
	if held {
		at.off()
		delete(attached, w)
	}
	return held
}

// dropAttachments removes this window from its parent and its children from it.
// The window's own teardown calls it.
func (w *WebviewWindow) dropAttachments() {
	if forget(w) {
		InvokeSync(func() { detachWindow(w) })
	}
	attachedMutex.Lock()
	children := []*WebviewWindow{}
	for child, at := range attached {
		if at.parent == w {
			children = append(children, child)
		}
	}
	attachedMutex.Unlock()
	for _, child := range children {
		forget(child)
		// The parent retains its children, so the relationship is ended here
		// rather than left for a window that is going away.
		child.Detach()
	}
}
