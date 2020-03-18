package main

import (
	"context"
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
	"goji.io"
	"goji.io/pat"
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

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalln("L38:", err)
	}
	addr := l.Addr().String()
	log.Println("L33:", addr)

	browser_process_handler.initial_url = flag.String("url", fmt.Sprintf("http://%s/html/wasm_exec.html", addr), "URL")
	flag.Parse()

	s := capi.NewCSettingsT()
	s.SetLogSeverity(capi.LogseverityWarning)
	s.SetNoSandbox(0)
	s.SetMultiThreadedMessageLoop(0)
	s.SetRemoteDebuggingPort(8088)
	cef.Initialize(mainArgs, s, app)

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/html/wasm_exec.js"), func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, runtime.GOROOT()+"/misc/wasm/wasm_exec.js")
	})
	mux.Handle(pat.Get("/html/*"), http.StripPrefix("/html", http.FileServer(http.Dir("./html"))))
	mux.Handle(pat.Get("/wasm/*"), http.StripPrefix("/wasm", http.FileServer(http.Dir("./wasm"))))

	srv := &http.Server{Handler: mux}

	go func() {
		if err := srv.Serve(l); err != http.ErrServerClosed {
			log.Fatalln("L50:", err)
		}
	}()

	capi.RunMessageLoop()
	defer capi.Shutdown()

	ctx := context.Background()
	srv.Shutdown(ctx)
}
func init() {
	var _ capi.OnBeforeCloseHandler = myLifeSpanHandler{}
}

type myLifeSpanHandler struct {
}

func (myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.Logf("L89:")
	capi.QuitMessageLoop()
}

func init() {
	var _ capi.OnContextInitializedHandler = myBrowserProcessHandler{}
}

type myBrowserProcessHandler struct {
	// this reference forms an UNgabagecollectable circular reference
	// To GC, call myBrowserProcessHandler.SetCBrowserProcessHandlerT(nil)
	capi.RefToCBrowserProcessHandlerT

	capi.RefToCClientT
	initial_url *string
}

func (bph myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("L108:")
	windowInfo := capi.NewCWindowInfoT()
	windowInfo.SetStyle(capi.WinWsOverlappedwindow | capi.WinWsClipchildren |
		capi.WinWsClipsiblings | capi.WinWsVisible)
	windowInfo.SetParentWindow(nil)
	windowInfo.SetX(capi.WinCwUseDefault)
	windowInfo.SetY(capi.WinCwUseDefault)
	windowInfo.SetWidth(capi.WinCwUseDefault)
	windowInfo.SetHeight(capi.WinCwUseDefault)
	windowInfo.SetWindowName("Cefingo Wasm http Example")

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
