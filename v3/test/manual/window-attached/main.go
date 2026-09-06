//go:build darwin

package main

import (
	"log"
	"net/http"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// A window placed over a point in another window's content.
//
// The main window draws a box. A second, frameless window is attached over that
// box: it moves with the main window and stays above it.
func main() {
	app := application.New(application.Options{
		Name: "attached window manual test",
		Assets: application.AssetOptions{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				if r.URL.Path == "/over" {
					_, _ = w.Write([]byte(over))
					return
				}
				_, _ = w.Write([]byte(page))
			}),
		},
	})

	below := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "attached window",
		Width:            700,
		Height:           460,
		BackgroundColour: application.NewRGB(20, 22, 30),
		URL:              "/",
	})

	over := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "over",
		Width:            300,
		Height:           200,
		Frameless:        true,
		Hidden:           true,
		BackgroundColour: application.NewRGB(12, 26, 20),
		URL:              "/over",
		Mac:              application.MacWindow{CornerRadius: 12, DisableShadow: true},
	})

	below.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		over.Show()
		// 40, 120 points from the top left of the main window's content.
		if err := over.Attach(below, 40, 120); err != nil {
			log.Fatal(err)
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

const page = `<!doctype html><meta charset=utf-8><title>host</title>
<style>
 body{margin:0;background:#14161e;color:#c7ccd8;font:13px/1.5 system-ui}
 p{padding:12px}
 #box{position:absolute;left:40px;top:120px;width:300px;height:200px;
      border:1px dashed #3a4256;box-sizing:border-box}
</style>
<p>The attached window sits over the dashed box. Move this window and it follows.</p>
<div id=box></div>`

const over = `<!doctype html><meta charset=utf-8><title>over</title>
<style>
 body{margin:0;background:#0c1a14;color:#7fe3b0;font:12px/1.6 ui-monospace,monospace}
 pre{padding:10px;margin:0}
</style>
<pre id=out></pre>
<script type="module">
 const out = document.getElementById("out");
 out.textContent = "attached window\n" + location.href + "\n";
 try {
   const runtime = await import("/wails/runtime.js");
   out.textContent += "runtime: loaded\n";
 } catch (why) {
   out.textContent += "runtime: " + why.message + "\n";
 }
</script>`
