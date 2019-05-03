package main

import (
	"context"
	"flag"
	"runtime"

	// "fmt"
	"log"
	"net/http"
	"os"

	"github.com/turutcrane/cefingo/capi"
)

func init() {
	// capi.Initialize(i.e. cef_initialize) and some function should be called on
	// the main application thread to initialize the CEF browser process
	runtime.LockOSThread()
	// prefix := fmt.Sprintf("[%d] ", os.Getpid())
	// capi.Logger = log.New(os.Stdout, prefix, log.LstdFlags)
	// capi.RefCountLogOutput(true)

}

func main() {
	go func() {
		ppid := os.Getppid()
		proc, _ := os.FindProcess(ppid)
		status, _ := proc.Wait()
		log.Println("Parent:", ppid, status)
		os.Exit(0)
	}()

	life_span_handler := capi.AllocCLifeSpanHandlerT().Bind(&myLifeSpanHandler{})

	browser_process_handler := myBrowserProcessHandler{}
	capi.AllocCBrowserProcessHandlerT().Bind(&browser_process_handler)
	defer browser_process_handler.SetCBrowserProcessHandlerT(nil)

	client := capi.AllocCClient().Bind(&myClient{})
	client.AssocLifeSpanHandler(life_span_handler)

	browser_process_handler.SetCClientT(client)

	app := capi.AllocCAppT().Bind(&myApp{})
	app.AssocBrowserProcessHandler(browser_process_handler.GetCBrowserProcessHandlerT())
	capi.ExecuteProcess(app)

	browser_process_handler.initial_url = flag.String("url", "https://www.golang.org/", "URL")
	flag.Parse()

	s := capi.Settings{}
	s.LogSeverity = capi.LogSeverityWarning
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	capi.Initialize(s, app)

	capi.RunMessageLoop()
	defer capi.Shutdown()

}

func addValueToContext(key interface{}, value interface{}) func(http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		mw := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, key, value)
			r = r.WithContext(ctx)
			inner.ServeHTTP(w, r)
		}
		return http.HandlerFunc(mw)
	}
}

type myLifeSpanHandler struct {
}

func (myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	// this reference forms an UNgabagecollectable circular reference
	// To GC, call myBrowserProcessHandler.SetCBrowserProcessHandlerT(nil)
	capi.RefToCBrowserProcessHandlerT

	capi.RefToCClientT
	initial_url *string
}

func (bph myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.BrowserHostCreateBrowser(
		"Cefingo Example",
		*bph.initial_url,
		bph.GetCClientT())
}

type myClient struct {
}

type myApp struct {
}
