//go:build darwin && !ios && !server

package application

/*
#cgo CFLAGS: -mmacosx-version-min=10.13 -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#include <stdlib.h>
#import "Cocoa/Cocoa.h"
#import <WebKit/WebKit.h>
#import "webview_window_darwin.h"

// Returns the webview the window was created with. Its configuration holds the
// scheme handler, the message handler and the data store, and its delegate
// handles navigation and script messages.
static WKWebView* hostWebview(void* nsWindow) {
	NSWindow<WailsWebviewWindow>* window = (NSWindow<WailsWebviewWindow>*)nsWindow;
	if (window == nil) return nil;
	return window.webView;
}

// Converts a rect measured from the content view's top left into an AppKit frame
// aligned to the display's pixels.
//
// A webview renders to a raster surface. A frame on a fraction of a pixel
// produces a blurred edge, and two adjacent webviews leave a gap between them.
// Aligning inward puts each edge on a pixel and does not enlarge the rect.
// The window must have a content view. Every caller checks, because a rect
// measured from a height of zero lands off screen.
static NSRect webviewAligned(NSWindow* window, double x, double y, double width, double height) {
	CGFloat top = [window contentView].bounds.size.height;
	NSRect want = NSMakeRect(x, top - y - height, width, height);
	return [window backingAlignedRect:want options:NSAlignAllEdgesInward];
}

// Adds a webview to the window's content view and returns it.
//
// It is created from the window's configuration, so it loads from the same
// scheme, calls the same bindings, receives the same events and shares one
// process pool. It has no autoresizing mask, because the caller sets its frame.
void* webviewNew(void* nsWindow, double x, double y, double width, double height,
		const char* url, int transparent, int hidden,
		double red, double green, double blue, double alpha) {
	WKWebView* host = hostWebview(nsWindow);
	if (host == nil) return NULL;
	NSWindow* window = (NSWindow*)nsWindow;
	NSView* content = [window contentView];
	if (content == nil) return NULL;

	// Autoreleased, as every view this package creates is: the content view
	// retains it in addSubview below and is then its only owner, so
	// removeFromSuperview releases it for good.
	WKWebView* view = [[[WKWebView alloc]
		initWithFrame:webviewAligned(window, x, y, width, height)
		configuration:host.configuration] autorelease];
	// The frame is given from the content view's top left, and an unflipped
	// content view measures from its bottom, so the distance to the top is what
	// is held. Without this the view moves down the window as the window grows.
	[view setAutoresizingMask:NSViewMinYMargin];
	// The UI delegate answers this view's own dialogs and file pickers. The
	// navigation delegate is not set: it reports what it sees under the window's
	// id, so an added webview's navigations would be reported as the window's.
	[view setUIDelegate:host.UIDelegate];
	[view setHidden:hidden ? YES : NO];

	// The colour displayed before the document renders. Public since macOS 12, so
	// the call is compiled only against an SDK that declares it.
#if MAC_OS_X_VERSION_MAX_ALLOWED >= 120000
	if (@available(macOS 12.0, *)) {
		view.underPageBackgroundColor =
			[NSColor colorWithSRGBRed:red green:green blue:blue alpha:alpha];
	}
#endif
	if (transparent) {
		// A webview renders only the area it has laid out and fills the rest with
		// opaque white. No public interface turns that off. The key below is set
		// by name, which this package already does for a transparent window.
		//
		// Setting an undeclared key raises, and a key that still resolves may no
		// longer drive compositing, so the value is set inside a guard and read
		// back. The caller is told when either step fails, because a webview
		// that keeps its background is not what was asked for.
		@try {
			[view setValue:@NO forKey:@"drawsBackground"];
			id now = [view valueForKey:@"drawsBackground"];
			if (![now isKindOfClass:[NSNumber class]] || [now boolValue]) {
				[view removeFromSuperview];
				return NULL;
			}
		} @catch (NSException* refused) {
			[view removeFromSuperview];
			return NULL;
		}
	}

	[content addSubview:view positioned:NSWindowAbove relativeTo:nil];
	if (url != NULL && url[0] != '\0') {
		NSURL* target = [NSURL URLWithString:[NSString stringWithUTF8String:url]];
		if (target != nil) [view loadRequest:[NSURLRequest requestWithURL:target]];
	}
	return view;
}

void webviewSetBounds(void* handle, double x, double y, double width, double height) {
	WKWebView* view = (WKWebView*)handle;
	NSWindow* window = [view window];
	if (window == nil || [window contentView] == nil) return;
	view.frame = webviewAligned(window, x, y, width, height);
}

void webviewSetHidden(void* handle, int hidden) {
	[(WKWebView*)handle setHidden:hidden ? YES : NO];
}

void webviewSetURL(void* handle, const char* url) {
	WKWebView* view = (WKWebView*)handle;
	NSURL* target = [NSURL URLWithString:[NSString stringWithUTF8String:url]];
	if (target != nil) [view loadRequest:[NSURLRequest requestWithURL:target]];
}

void webviewExecJS(void* handle, const char* js) {
	[(WKWebView*)handle evaluateJavaScript:[NSString stringWithUTF8String:js]
						 completionHandler:nil];
}

void webviewDestroy(void* handle) {
	[(WKWebView*)handle removeFromSuperview];
}
*/
import "C"

import "unsafe"

// macosWebview is a webview added to a window's content view.
type macosWebview struct {
	handle unsafe.Pointer
}

func addWebview(window *WebviewWindow, options WebviewOptions) (webviewImpl, error) {
	nsWindow := window.impl.nativeWindow()
	if nsWindow == nil {
		return nil, ErrWebviewWindowGone
	}
	target := C.CString(options.URL)
	defer C.free(unsafe.Pointer(target))

	handle := C.webviewNew(nsWindow,
		C.double(options.X), C.double(options.Y),
		C.double(options.Width), C.double(options.Height),
		target, boolToCInt(options.Transparent), boolToCInt(options.Hidden),
		C.double(float64(options.BackgroundColour.Red)/255),
		C.double(float64(options.BackgroundColour.Green)/255),
		C.double(float64(options.BackgroundColour.Blue)/255),
		C.double(float64(options.BackgroundColour.Alpha)/255))
	if handle == nil {
		// Transparent was asked for and the key that turns the background off is
		// gone. Reported rather than ignored: the webview would fill everything
		// it has not laid out with opaque white.
		if options.Transparent {
			return nil, ErrWebviewNotTransparent
		}
		// The window has no content view, or it has no webview of its own to
		// take a configuration from. Both mean the window is not one this can
		// add to.
		return nil, ErrWebviewWindowGone
	}
	return &macosWebview{handle: handle}, nil
}

func (v *macosWebview) setBounds(x, y, width, height float64) {
	C.webviewSetBounds(v.handle, C.double(x), C.double(y), C.double(width), C.double(height))
}

func (v *macosWebview) setHidden(hidden bool) {
	C.webviewSetHidden(v.handle, boolToCInt(hidden))
}

func (v *macosWebview) setURL(url string) {
	target := C.CString(url)
	defer C.free(unsafe.Pointer(target))
	C.webviewSetURL(v.handle, target)
}

func (v *macosWebview) execJS(js string) {
	line := C.CString(js)
	defer C.free(unsafe.Pointer(line))
	C.webviewExecJS(v.handle, line)
}

func (v *macosWebview) nativeView() unsafe.Pointer { return v.handle }

func (v *macosWebview) destroy() { C.webviewDestroy(v.handle) }

func boolToCInt(on bool) C.int {
	if on {
		return 1
	}
	return 0
}
