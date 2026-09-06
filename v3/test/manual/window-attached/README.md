# A window attached over another window's content, manual test

Run on macOS from `v3`:

```sh
go run ./test/manual/window-attached
```

A window that presents part of the document below it, such as a modal or a
picker, must be positioned inside that document, move with it and stay above it.
A window positioned by screen coordinates alone does not.

`(*WebviewWindow).Attach(parent, x, y)` positions it over the parent's content,
x and y points from its top left, and makes it a child of the parent.

## Checks

1. The attached window covers the dashed box exactly.
2. Move the main window. The attached window moves with it.
3. Activate another application. The attached window stays above its parent.
4. It prints `runtime: loaded`: it is an ordinary window of this application and
   uses the same asset server and runtime.

Only macOS is implemented. `window_attach_other.go` states what Windows and
Linux would do.
