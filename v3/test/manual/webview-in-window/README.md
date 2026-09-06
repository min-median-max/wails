# A webview added to a window, manual test

Run on macOS from `v3`:

```sh
go run ./test/manual/webview-in-window
```

A window creates one webview and resizes it to fill the window. An application
that composites its own views over a page needs more than one webview in the
same window: a terminal, a browser or a preview, each a separate document, each
placed by the application, and each drawn by the system rather than by the page,
which cannot draw over them.

`(*WebviewWindow).AddWebview` adds one. It is created from the window's
configuration, so it loads from the same scheme, calls the same bindings and
receives the same events. The caller sets its position and size.

## Checks

1. Two boxes are drawn by the window's page and a webview covers each one. Both
   load `wails://localhost/inner`, which this application serves; a webview
   created without the window's configuration cannot load it.
2. Each webview prints its own name, so the two are separate documents.
3. Each prints `runtime: loaded` and a tick that increments every second. The
   runtime is imported from the same asset server and the tick is an event
   emitted to the window.
4. The window's page is drawn behind them. A page cannot draw over a webview the
   system composites, which is why the boxes are outlines.
5. Resize the window. The added webviews keep the position given from the
   window's top left, and keep their size.

Only macOS is implemented. `AddWebview` returns `ErrWebviewUnsupported`
elsewhere.
