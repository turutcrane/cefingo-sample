package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/turutcrane/cefingo"
)

var initial_url *string

func init() {
	prefix := fmt.Sprintf("[%d] ", os.Getpid())
	log.SetOutput(os.Stdout)
	log.SetPrefix(prefix)

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
	cLifeSpanHandler := cefingo.AllocCLifeSpanHandler(&life_span_handler)

	browser_process_handler := myBrowserProcessHandler{}
	cBrowserProcessHandler := cefingo.AllocCBrowserProcessHandler(&browser_process_handler)

	client := myClient{}
	cefClient = cefingo.AllocCClient(&client)
	cefingo.AssocLifeSpanHandler(cefClient, cLifeSpanHandler)

	app := myApp{}
	cefApp := cefingo.AllocCApp(&app)
	cefingo.AssocBrowserProcessHandler(cefApp, cBrowserProcessHandler)
	cefingo.ExecuteProcess(cefApp)

	initial_url = flag.String("url", "https://www.google.com/", "URL")
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
