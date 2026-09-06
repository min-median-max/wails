//go:build !(darwin && !ios && !server)

package application

// Only macOS is implemented. On Windows this would create a second WebView2
// controller on the window's HWND; on Linux it would add a second WebKitGTK view
// to the window's container.
func addWebview(_ *WebviewWindow, _ WebviewOptions) (webviewImpl, error) {
	return nil, ErrWebviewUnsupported
}
