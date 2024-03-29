package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/turutcrane/cefingo/capi"
	"github.com/turutcrane/cefingo/cef"
	"github.com/turutcrane/win32api"
)

func init() {
	// capi.Initialize(i.e. cef_initialize) and some function should be called on
	// the main application thread to initialize the CEF browser process
	// runtime.LockOSThread()
	prefix := fmt.Sprintf("[%d] ", os.Getpid())
	capi.Logger = log.New(os.Stdout, prefix, log.LstdFlags)
	capi.RefCountLogOutput(true)
	capi.RefCountLogTrace(true)

}

type cefProcessType int

const (
	OtherProcces cefProcessType = iota
	BrowserProcess
	RenderProcess
)

func main() {

	// go func() {
	// 	ppid := os.Getppid()
	// 	proc, _ := os.FindProcess(ppid)
	// 	status, _ := proc.Wait()
	// 	log.Println("Parent:", ppid, status)
	// 	os.Exit(0)
	// }()

	mainArgs := capi.NewCMainArgsT()
	cef.CMainArgsTSetInstance(mainArgs)

	doCef(mainArgs)
	runtime.GC()

	capi.Logln("T73: End of doCef ======================================")

	if !capi.CurrentlyOn(capi.TidUi) {
		log.Println("T37: No UiThead")
	}
	capi.Shutdown()

	capi.Logf("T71:===")
	time.Sleep(2 * time.Second)
}

func doCef(mainArgs *capi.CMainArgsT) {

	s := capi.NewCSettingsT()
	s.SetLogSeverity(capi.LogseverityWarning)
	s.SetNoSandbox(true)
	s.SetMultiThreadedMessageLoop(false)
	s.SetRemoteDebuggingPort(8088)

	app := &myApp{}
	app.app = capi.NewCAppT(app)
	defer app.app.Unref()

	client := &myClient{}
	client.client = capi.NewCClientT(client)
	defer client.client.Unref() // UnbindAll()

	client.lifeSpanHandler = capi.NewCLifeSpanHandlerT(client)
	defer client.lifeSpanHandler.Unref() // .UnbindAll()

	app.browserProcessHandler = capi.NewCBrowserProcessHandlerT(app)
	defer app.browserProcessHandler.Unref() // .UnbindAll()

	// app.SetCClientT(client.GetCClientT())
	// defer app.SetCClientT(nil)
	app.client = client

	cef.ExecuteProcess(mainArgs, app.app)

	app.initial_url = flag.String("url", "https://www.golang.org/", "URL")
	flag.Parse() // Be after cef.ExecuteProcess or implement cef_browser_process_handler::on_before_child_process_launch

	cef.Initialize(mainArgs, s, app.app)

	capi.RunMessageLoop()
	if !capi.CurrentlyOn(capi.TidUi) {
		log.Println("T148: No UiThead")
	}
	runtime.LockOSThread()
}

func (app *myApp) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	windowInfo := capi.NewCWindowInfoT()
	windowInfo.SetStyle(win32api.WsOverlappedwindow | win32api.WsClipchildren |
		win32api.WsClipsiblings | win32api.WsVisible)
	windowInfo.SetParentWindow(nil)
	bound := capi.NewCRectT()
	bound.SetX(win32api.CwUsedefault)
	bound.SetY(win32api.CwUsedefault)
	bound.SetWidth(win32api.CwUsedefault)
	bound.SetHeight(win32api.CwUsedefault)
	windowInfo.SetBounds(*bound)
	windowInfo.SetWindowName("Cefingo Simple Example")

	browserSettings := capi.NewCBrowserSettingsT()

	if !capi.CurrentlyOn(capi.TidUi) {
		capi.Panicln("T160: Not on UI")
	}

	capi.BrowserHostCreateBrowser(windowInfo,
		app.client.client,
		*app.initial_url,
		browserSettings, nil, nil)
}

type myClient struct {
	client          *capi.CClientT
	lifeSpanHandler *capi.CLifeSpanHandlerT
}

func init() {
	var client *myClient
	var _ capi.GetLifeSpanHandlerHandler = client

	// LifeSpanHandler
	var _ capi.OnBeforeCloseHandler = client
}

func (client *myClient) GetLifeSpanHandler(self *capi.CClientT) *capi.CLifeSpanHandlerT {
	return client.lifeSpanHandler
}

func (client *myClient) OnBeforeClose(self *capi.CLifeSpanHandlerT, browser *capi.CBrowserT) {
	if capi.CurrentlyOn(capi.TidUi) {
		capi.Logf("T172:-----------------------------")
	} else {
		capi.Panicln("T205: Not Ui Thread")
	}

	capi.QuitMessageLoop()
}

type myApp struct {
	app                   *capi.CAppT
	browserProcessHandler *capi.CBrowserProcessHandlerT
	// capi.RefToCClientT
	client      *myClient
	initial_url *string
}

func init() {
	var app *myApp
	// CAppT
	var _ capi.GetBrowserProcessHandlerHandler = app

	// CBrowserProcessHandlerT
	var _ capi.OnContextInitializedHandler = app
}

func (app *myApp) GetBrowserProcessHandler(self *capi.CAppT) *capi.CBrowserProcessHandlerT {
	return app.browserProcessHandler
}
