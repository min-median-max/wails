//go:build !(darwin && !ios && !server)

package application

// Only macOS is implemented. On Windows this would set the parent as the owner
// window and reposition on the parent's WM_MOVE; on Linux it would call
// gtk_window_set_transient_for and reposition on the parent's configure-event.
const attachSupported = false

func attachWindow(_ *WebviewWindow, _ *WebviewWindow, _, _ float64) {}

func detachWindow(_ *WebviewWindow) {}
