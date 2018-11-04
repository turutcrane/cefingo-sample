package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/turutcrane/cefingo"
	"goji.io"
	"goji.io/pat"
)

var initial_url *string

func init() {
	prefix := fmt.Sprintf("[%d] ", os.Getpid())
	log.SetOutput(os.Stdout)
	log.SetPrefix(prefix)

}

var cefClient *cefingo.CClientT

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
	// cefingo.RefCountLogOutput(true)
	// cefingo.LogOutput(true)

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

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalln("L38:", err)
	}
	addr := l.Addr().String()
	log.Println("L33:", addr)

	initial_url = flag.String("url", fmt.Sprintf("http://%s/html/wasm_exec.html", addr), "URL")
	flag.Parse()

	s := cefingo.Settings{}
	s.LogSeverity = cefingo.LogSeverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	cefingo.Initialize(s, cefApp)

	mux := goji.NewMux()
	mux.Handle(pat.Get("/html/*"), http.StripPrefix("/html", http.FileServer(http.Dir("./html"))))
	mux.Handle(pat.Get("/wasm/*"), http.StripPrefix("/wasm", http.FileServer(http.Dir("./wasm"))))

	srv := &http.Server{Handler: mux}

	go func() {
		if err := srv.Serve(l); err != http.ErrServerClosed {
			log.Fatalln("L50:", err)
		}
	}()

	cefingo.RunMessageLoop()
	defer cefingo.Shutdown()

	ctx := context.Background()
	srv.Shutdown(ctx)
}

type myLifeSpanHandler struct {
	cefingo.DefaultLifeSpanHandler
}

func (*myLifeSpanHandler) OnBeforeClose(self *cefingo.CLifeSpanHandlerT, brwoser *cefingo.CBrowserT) {
	cefingo.Logf("L89:")
	cefingo.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	cefingo.DefaultBrowserProcessHandler
}

func (*myBrowserProcessHandler) OnContextInitialized(sef *cefingo.CBrowserProcessHandlerT) {
	cefingo.Logf("L108:")
	cefingo.BrowserHostCreateBrowser("Cefingo Example", *initial_url, cefClient)
}

type myClient struct {
	cefingo.DefaultClient
}

type myApp struct {
	cefingo.DefaultApp
}
