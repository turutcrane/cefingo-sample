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

var initial_url *string

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

	life_span_handler := myLifeSpanHandler{}
	capi.AllocCLifeSpanHandlerT().Bind(&life_span_handler)

	browser_process_handler := myBrowserProcessHandler{}
	capi.AllocCBrowserProcessHandlerT().Bind(&browser_process_handler)

	client := myClient{}
	capi.AllocCClient().Bind(&client)
	client.GetCClientT().AssocLifeSpanHandler(life_span_handler.GetCLifeSpanHandlerT())

	browser_process_handler.SetCClientT(client.GetCClientT())

	app := myApp{}
	capi.AllocCAppT().Bind(&app)
	app.GetCAppT().AssocBrowserProcessHandler(browser_process_handler.GetCBrowserProcessHandlerT())
	capi.ExecuteProcess(app.GetCAppT())

	initial_url = flag.String("url", "https://www.golang.org/", "URL")
	flag.Parse()

	s := capi.Settings{}
	s.LogSeverity = capi.LogSeverityWarning
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	capi.Initialize(s, app.GetCAppT())

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
	capi.RefToCLifeSpanHandlerT
}

func (*myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	capi.RefToCBrowserProcessHandlerT
	capi.RefToCClientT
}

func (bph *myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.BrowserHostCreateBrowser(
		"Cefingo Example",
		*initial_url,
		bph.GetCClientT())
}

type myClient struct {
	capi.RefToCClientT
}

type myApp struct {
	capi.RefToCAppT
}
