package main

import (
	"context"
	"flag"
	"runtime"

	// "fmt"
	"log"
	"net/http"
	"os"

	"github.com/turutcrane/cefingo"
)

var initial_url *string

func init() {
	// cefingo.Initialize(i.e. cef_initialize) and some function should be called on
	// the main application thread to initialize the CEF browser process
	runtime.LockOSThread()
	// prefix := fmt.Sprintf("[%d] ", os.Getpid())
	// cefingo.Logger = log.New(os.Stdout, prefix, log.LstdFlags)
	// cefingo.RefCountLogOutput(true)

}

var cefClient *cefingo.CClientT

func main() {
	go func() {
		ppid := os.Getppid()
		proc, _ := os.FindProcess(ppid)
		status, _ := proc.Wait()
		log.Println("Parent:", ppid, status)
		os.Exit(0)
	}()

	life_span_handler := myLifeSpanHandler{}
	cLifeSpanHandler := cefingo.AllocCLifeSpanHandlerT(&life_span_handler)

	browser_process_handler := myBrowserProcessHandler{}
	cBrowserProcessHandler := cefingo.AllocCBrowserProcessHandlerT(&browser_process_handler)

	client := myClient{}
	cefClient = cefingo.AllocCClient(&client)
	cefClient.AssocLifeSpanHandler(cLifeSpanHandler)

	app := myApp{}
	cefApp := cefingo.AllocCAppT(&app)
	cefApp.AssocBrowserProcessHandler(cBrowserProcessHandler)
	cefingo.ExecuteProcess(cefApp)

	initial_url = flag.String("url", "https://www.golang.org/", "URL")
	flag.Parse()

	s := cefingo.Settings{}
	s.LogSeverity = cefingo.LogSeverityWarning
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	cefingo.Initialize(s, cefApp)

	cefingo.RunMessageLoop()
	defer cefingo.Shutdown()

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
	cefingo.DefaultLifeSpanHandler
}

func (*myLifeSpanHandler) OnBeforeClose(self *cefingo.CLifeSpanHandlerT, brwoser *cefingo.CBrowserT) {
	cefingo.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	cefingo.DefaultBrowserProcessHandler
}

func (*myBrowserProcessHandler) OnContextInitialized(sef *cefingo.CBrowserProcessHandlerT) {
	cefingo.BrowserHostCreateBrowser("Cefingo Example", *initial_url, cefClient)
}

type myClient struct {
	cefingo.DefaultClient
}

type myApp struct {
	cefingo.DefaultApp
}
