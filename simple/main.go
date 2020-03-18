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
	"github.com/turutcrane/cefingo/cef"
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

	mainArgs := capi.NewCMainArgsT()
	mainArgs.SetWinHandle()

	life_span_handler := capi.AllocCLifeSpanHandlerT().Bind(&myLifeSpanHandler{})

	browser_process_handler := myBrowserProcessHandler{}
	capi.AllocCBrowserProcessHandlerT().Bind(&browser_process_handler)
	defer browser_process_handler.SetCBrowserProcessHandlerT(nil)

	client := capi.AllocCClientT().Bind(&myClient{})
	client.AssocLifeSpanHandlerT(life_span_handler)

	browser_process_handler.SetCClientT(client)

	app := capi.AllocCAppT().Bind(&myApp{})
	app.AssocBrowserProcessHandlerT(browser_process_handler.GetCBrowserProcessHandlerT())
	cef.ExecuteProcess(mainArgs, app)

	browser_process_handler.initial_url = flag.String("url", "https://www.golang.org/", "URL")
	flag.Parse()

	s := capi.NewCSettingsT()
	s.SetLogSeverity(capi.LogseverityWarning)
	s.SetNoSandbox(0)
	s.SetMultiThreadedMessageLoop(0)
	s.SetRemoteDebuggingPort(8088)
	cef.Initialize(mainArgs, s, app)

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
	windowInfo := capi.NewCWindowInfoT()
	windowInfo.SetStyle(capi.WinWsOverlappedwindow | capi.WinWsClipchildren |
		capi.WinWsClipsiblings | capi.WinWsVisible)
	windowInfo.SetParentWindow(nil)
	windowInfo.SetX(capi.WinCwUseDefault)
	windowInfo.SetY(capi.WinCwUseDefault)
	windowInfo.SetWidth(capi.WinCwUseDefault)
	windowInfo.SetHeight(capi.WinCwUseDefault)
	windowInfo.SetWindowName("Cefingo Simple Example")

	browserSettings := capi.NewCBrowserSettingsT()

	capi.BrowserHostCreateBrowser(windowInfo,
		bph.GetCClientT(),
		*bph.initial_url,
		browserSettings, nil, nil)
}

type myClient struct {
}

type myApp struct {
}
