package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/turutcrane/cefingo/capi"
	"github.com/turutcrane/cefingo/cef"
	"github.com/turutcrane/win32api"
	"goji.io"
	"goji.io/pat"
)

//go:embed html
var htmlFs embed.FS

//go:embed wasm
var wasmFs embed.FS

func init() {
	// capi.Initialize(i.e. cef_initialize) and some function should be called on
	// the main application thread to initialize the CEF browser process
	runtime.LockOSThread()
	// prefix := fmt.Sprintf("[%d] ", os.Getpid())
	// capi.Logger = log.New(os.Stdout, prefix, log.LstdFlags)
	// capi.RefCountLogOutput(true)

}

func main() {
	defer log.Println("L31: Graceful Shutdowned")
	log.Println("L33: started:", "Pid:", os.Getpid(), "PPid:", os.Getppid(), os.Args)
	// Exit when parant (go command) is exited.
	go func() {
		ppid := os.Getppid()
		proc, _ := os.FindProcess(ppid)
		status, _ := proc.Wait()
		log.Println("Parent:", ppid, status)
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}()

	mainArgs := capi.NewCMainArgsT()
	cef.CMainArgsTSetInstance(mainArgs)

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/html/wasm_exec.js"), func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, runtime.GOROOT()+"/misc/wasm/wasm_exec.js")
	})

	mux.Handle(pat.Get("/html/*"), http.FileServer(http.FS(htmlFs)))
	mux.Handle(pat.Get("/wasm/*"), http.FileServer(http.FS(wasmFs)))

	srv := &http.Server{Handler: mux}

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalln("L38:", err)
	}

	go func() {
		if err := srv.Serve(l); err != http.ErrServerClosed {
			log.Fatalln("L50:", err)
		}
	}()

	doCef(mainArgs, l.Addr().String())
	runtime.GC()

	capi.Shutdown()

	ctx := context.Background()
	srv.Shutdown(ctx)
}

func doCef(mainArgs *capi.CMainArgsT, addr string) {
	client := &myClient{}
	client.client = capi.NewCClientT(client)
	defer client.client.Unref() // .UnbindAll()

	client.lifeSpanHandler =  capi.NewCLifeSpanHandlerT(client)
	defer client.lifeSpanHandler.Unref() // .UnbindAll()

	app := &myApp{}
	app.app = capi.NewCAppT(app)
	defer app.app.Unref() //.UnbindAll()
	app.myBrowserProcessHandler.client = client

	app.browserProcessHandler = capi.NewCBrowserProcessHandlerT(app)
	defer app.browserProcessHandler.Unref() // .UnbindAll()

	cef.ExecuteProcess(mainArgs, app.app)

	app.initial_url = flag.String("url", fmt.Sprintf("http://%s/html/wasm_exec.html", addr), "URL")
	flag.Parse()

	s := capi.NewCSettingsT()
	s.SetLogSeverity(capi.LogseverityWarning)
	s.SetNoSandbox(false)
	s.SetMultiThreadedMessageLoop(false)
	s.SetRemoteDebuggingPort(8088)
	cef.Initialize(mainArgs, s, app.app)

	capi.RunMessageLoop()
	capi.Logln("T112:")

}

func init() {
	// capi.CLifeSpanHandlerT handler
	var _ capi.OnBeforeCloseHandler = (*myClient)(nil)
}

func (*myClient) OnBeforeClose(self *capi.CLifeSpanHandlerT, browser *capi.CBrowserT) {
	capi.Logf("L89:")
	capi.QuitMessageLoop()
}

func init() {
	var _ capi.OnContextInitializedHandler = (*myBrowserProcessHandler)(nil)
}

type myBrowserProcessHandler struct {
	// this reference forms an UNgabagecollectable circular reference
	// To GC, call myBrowserProcessHandler.GetCBrowserProcessHandlerT().UnbindAll()
	browserProcessHandler  *capi.CBrowserProcessHandlerT

	client *myClient
	// capi.RefToCClientT
	initial_url *string
}

func (bph *myBrowserProcessHandler) GetBrowserProcessHandler(*capi.CAppT) *capi.CBrowserProcessHandlerT {
	return bph.browserProcessHandler
}

func (bph *myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("L108:")
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
	windowInfo.SetWindowName("Cefingo Wasm http Example")

	browserSettings := capi.NewCBrowserSettingsT()

	capi.BrowserHostCreateBrowser(windowInfo,
		bph.client.client,
		*bph.initial_url,
		browserSettings, nil, nil)
}

type myClient struct {
	client *capi.CClientT
	lifeSpanHandler *capi.CLifeSpanHandlerT
}

func init() {
	var _ capi.GetLifeSpanHandlerHandler = (*myClient)(nil)
}

func (client *myClient) GetLifeSpanHandler(c *capi.CClientT) *capi.CLifeSpanHandlerT {
	return client.lifeSpanHandler
}

type myApp struct {
	app *capi.CAppT
	myBrowserProcessHandler
}

func init() {
	var _ capi.GetBrowserProcessHandlerHandler = (*myApp)(nil)
}
