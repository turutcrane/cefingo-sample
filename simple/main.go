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
	capi.AllocCAppT().Bind(app)
	defer app.GetCAppT().UnbindAll()

	client := &myClient{}
	capi.AllocCClientT().Bind(client)
	defer client.GetCClientT().UnbindAll()

	capi.AllocCLifeSpanHandlerT().Bind(client)
	defer client.GetCLifeSpanHandlerT().UnbindAll()

	capi.AllocCBrowserProcessHandlerT().Bind(app)
	defer app.GetCBrowserProcessHandlerT().UnbindAll()

	app.SetCClientT(client.GetCClientT())
	defer app.SetCClientT(nil)

	cef.ExecuteProcess(mainArgs, app.GetCAppT())

	app.initial_url = flag.String("url", "https://www.golang.org/", "URL")
	flag.Parse() // Be after cef.ExecuteProcess or implement cef_browser_process_handler::on_before_child_process_launch

	cef.Initialize(mainArgs, s, app.GetCAppT())

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
	windowInfo.SetX(win32api.CwUsedefault)
	windowInfo.SetY(win32api.CwUsedefault)
	windowInfo.SetWidth(win32api.CwUsedefault)
	windowInfo.SetHeight(win32api.CwUsedefault)
	windowInfo.SetWindowName("Cefingo Simple Example")

	browserSettings := capi.NewCBrowserSettingsT()

	if !capi.CurrentlyOn(capi.TidUi) {
		capi.Panicln("T160: Not on UI")
	}

	capi.BrowserHostCreateBrowser(windowInfo,
			app.GetCClientT(),
			*app.initial_url,
			browserSettings, nil, nil)
}

type myClient struct {
	capi.RefToCClientT
	capi.RefToCLifeSpanHandlerT
}

func init() {
	var client *myClient
	var _ capi.GetLifeSpanHandlerHandler = client

	// LifeSpanHandler
	var _ capi.OnBeforeCloseHandler = client
}

func (client *myClient) GetLifeSpanHandler(self *capi.CClientT) *capi.CLifeSpanHandlerT {
	return client.GetCLifeSpanHandlerT()
}

func (client *myClient) OnBeforeClose(self *capi.CLifeSpanHandlerT, browser *capi.CBrowserT) {
	defer browser.ForceUnref()
	if capi.CurrentlyOn(capi.TidUi) {
		capi.Logf("T172:-----------------------------")
	} else {
		capi.Panicln("T205: Not Ui Thread")
	}

	capi.QuitMessageLoop()
}

type myApp struct {
	capi.RefToCAppT
	capi.RefToCBrowserProcessHandlerT
	capi.RefToCClientT

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
	return app.GetCBrowserProcessHandlerT()
}
