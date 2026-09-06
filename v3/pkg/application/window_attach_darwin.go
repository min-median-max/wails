//go:build darwin && !ios && !server

package application

/*
#cgo CFLAGS: -mmacosx-version-min=10.13 -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import "Cocoa/Cocoa.h"

// Positions the child over the parent's content and makes it a child window.
//
// A child window keeps its offset from the parent when the parent moves, and the
// window server draws it above the parent. AppKit computes that offset from the
// child's frame, so the frame is set before the window is attached.
void windowAttach(void* childWindow, void* parentWindow, double x, double y) {
	NSWindow* child = (NSWindow*)childWindow;
	NSWindow* parent = (NSWindow*)parentWindow;
	if (child == nil || parent == nil) return;

	NSView* content = [parent contentView];
	if (content == nil) return;

	// The point is measured from the top left of the parent's content view.
	// AppKit measures from the bottom left.
	NSRect inContent = NSMakeRect(x, content.bounds.size.height - y - child.frame.size.height,
		child.frame.size.width, child.frame.size.height);
	NSRect inWindow = [content convertRect:inContent toView:nil];
	NSRect onScreen = [parent convertRectToScreen:inWindow];
	[child setFrameOrigin:onScreen.origin];

	if ([child parentWindow] != parent) {
		[[child parentWindow] removeChildWindow:child];
		[parent addChildWindow:child ordered:NSWindowAbove];
	}
}

void windowDetach(void* childWindow) {
	NSWindow* child = (NSWindow*)childWindow;
	NSWindow* parent = [child parentWindow];
	if (parent != nil) [parent removeChildWindow:child];
}
*/
import "C"

const attachSupported = true

func attachWindow(child *WebviewWindow, parent *WebviewWindow, x, y float64) {
	C.windowAttach(child.impl.nativeWindow(), parent.impl.nativeWindow(),
		C.double(x), C.double(y))
}

func detachWindow(child *WebviewWindow) {
	C.windowDetach(child.impl.nativeWindow())
}
