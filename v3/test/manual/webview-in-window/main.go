//go:build darwin

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// A window with two webviews added to it, beside the webview it was created
// with.
//
// The page fills the window and draws two boxes. Each added webview is placed
// over a box and loads a page this application serves, which shows that it uses
// the same scheme as the window's own webview.
func main() {
	app := application.New(application.Options{
		Name: "webview in window manual test",
		Assets: application.AssetOptions{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				switch r.URL.Path {
				case "/inner":
					_, _ = w.Write([]byte(inner))
				default:
					_, _ = w.Write([]byte(page))
				}
			}),
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "webview in window",
		Width:            900,
		Height:           520,
		BackgroundColour: application.NewRGB(20, 22, 30),
		URL:              "/",
	})

	// The window's own webview must exist before another is added to it.
	window.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		add(window, 40, 80, "left")
		add(window, 480, 80, "right")
	})

	// One event to the window every second. An added webview runs the same
	// runtime and subscribes to the same events, so it receives them.
	go func() {
		for count := 1; ; count++ {
			time.Sleep(time.Second)
			window.EmitEvent("tick", count)
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func add(window *application.WebviewWindow, x, y float64, name string) {
	view, err := window.AddWebview(application.WebviewOptions{
		URL:              "/inner?name=" + name,
		X:                x,
		Y:                y,
		Width:            380,
		Height:           360,
		BackgroundColour: application.NewRGB(12, 26, 20),
		Transparent:      true,
	})
	if err != nil {
		log.Printf("%s: %v", name, err)
		return
	}
	log.Printf("%s: added, native view %p", name, view.NativeView())
}

const page = `<!doctype html><meta charset=utf-8><title>host</title>
<style>
 body{margin:0;background:#14161e;color:#c7ccd8;font:13px/1.5 system-ui}
 .box{position:absolute;width:380px;height:360px;top:80px;
      border:1px solid #3a4256;box-sizing:border-box}
 #a{left:40px} #b{left:480px}
 p{padding:12px}
</style>
<p>Two webviews are added over the boxes below. Each opens /inner from this
application's own assets.</p>
<div class=box id=a></div><div class=box id=b></div>`

const inner = `<!doctype html><meta charset=utf-8><title>inner</title>
<style>
 body{margin:0;background:#0c1a14;color:#7fe3b0;font:12px/1.6 ui-monospace,monospace}
 pre{padding:10px;margin:0}
</style>
<pre id=out></pre>
<script type="module">
 const name = new URLSearchParams(location.search).get("name");
 const out = document.getElementById("out");
 const say = (line) => { out.textContent += line + "\n"; };
 say(name);
 say(location.href);
 try {
   const runtime = await import("/wails/runtime.js");
   say("runtime: loaded");
   runtime.Events.On("tick", (event) => {
     out.textContent = name + "\n" + location.href + "\n" +
       "runtime: loaded\ntick " + event.data;
   });
 } catch (why) {
   say("runtime: " + why.message);
 }
</script>`
