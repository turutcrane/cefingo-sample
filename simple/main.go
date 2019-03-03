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

var cefClient *capi.CClientT

func main() {
	go func() {
		ppid := os.Getppid()
		proc, _ := os.FindProcess(ppid)
		status, _ := proc.Wait()
		log.Println("Parent:", ppid, status)
		os.Exit(0)
	}()

	life_span_handler := myLifeSpanHandler{}
	cLifeSpanHandler := capi.AllocCLifeSpanHandlerT(&life_span_handler)

	browser_process_handler := myBrowserProcessHandler{}
	cBrowserProcessHandler := capi.AllocCBrowserProcessHandlerT(&browser_process_handler)

	client := myClient{}
	cefClient = capi.AllocCClient(&client)
	cefClient.AssocLifeSpanHandler(cLifeSpanHandler)

	app := myApp{}
	cefApp := capi.AllocCAppT(&app)
	cefApp.AssocBrowserProcessHandler(cBrowserProcessHandler)
	capi.ExecuteProcess(cefApp)

	initial_url = flag.String("url", "https://www.golang.org/", "URL")
	flag.Parse()

	s := capi.Settings{}
	s.LogSeverity = capi.LogSeverityWarning
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	capi.Initialize(s, cefApp)

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
	capi.DefaultLifeSpanHandler
}

func (*myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	capi.DefaultBrowserProcessHandler
}

func (*myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.BrowserHostCreateBrowser("Cefingo Example", *initial_url, cefClient)
}

type myClient struct {
	capi.DefaultClient
}

type myApp struct {
	capi.DefaultApp
}
