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
	"goji.io"
	"goji.io/pat"
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

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalln("L38:", err)
	}
	addr := l.Addr().String()
	log.Println("L33:", addr)

	initial_url = flag.String("url", fmt.Sprintf("http://%s/html/wasm_exec.html", addr), "URL")
	flag.Parse()

	s := capi.Settings{}
	s.LogSeverity = capi.LogSeverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	capi.Initialize(s, cefApp)

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/html/wasm_exec.js"), func (w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, runtime.GOROOT() + "/misc/wasm/wasm_exec.js")
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

type myLifeSpanHandler struct {
	capi.DefaultLifeSpanHandler
}

func (*myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.Logf("L89:")
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	capi.DefaultBrowserProcessHandler
}

func (*myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("L108:")
	capi.BrowserHostCreateBrowser("Cefingo Example", *initial_url, cefClient)
}

type myClient struct {
	capi.DefaultClient
}

type myApp struct {
	capi.DefaultApp
}
