package main

import (
	"bytes"
	"runtime"
	// "fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"time"
	"unsafe"

	"github.com/turutcrane/cefingo/capi"
	"github.com/vincent-petithory/dataurl"
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
	// defer log.Println("L31: Graceful Shutdowned")
	// log.Println("L33: started:", "Pid:", os.Getpid(), "PPid:", os.Getppid(), os.Args)
	// Exit when parant (go command) is exited.
	go func() {
		ppid := os.Getppid()
		proc, _ := os.FindProcess(ppid)
		status, _ := proc.Wait()
		log.Println("Parent:", ppid, status)
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}()

	life_span_handler := capi.AllocCLifeSpanHandlerT().Bind(&myLifeSpanHandler{})

	browser_process_handler := myBrowserProcessHandler{}
	capi.AllocCBrowserProcessHandlerT().Bind(&browser_process_handler)
	defer browser_process_handler.SetCBrowserProcessHandlerT(nil)

	client := capi.AllocCClient().Bind(&myClient{})
	client.AssocLifeSpanHandler(life_span_handler)
	browser_process_handler.SetCClientT(client)

	app := capi.AllocCAppT().Bind(&myApp{})
	app.AssocBrowserProcessHandler(browser_process_handler.GetCBrowserProcessHandlerT())

	render_process_handler := capi.AllocCRenderProcessHandlerT().Bind(&myRenderProcessHander{})
	app.AssocRenderProcessHandler(render_process_handler)

	capi.ExecuteProcess(app)

	html, err := makeHtlmString()
	if err != nil {
		log.Panicln("Can not open html")
	}
	durl := dataurl.New(html, "text/html").String()
	browser_process_handler.initial_url = durl

	s := capi.Settings{}
	s.LogSeverity = capi.LogSeverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	capi.Initialize(s, app)

	capi.RunMessageLoop()
	defer capi.Shutdown()

}

type myLifeSpanHandler struct {
}

func (myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.Logf("L89:")
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	// this reference forms an UNgabagecollectable circular reference
	// To GC, call myBrowserProcessHandler.SetCBrowserProcessHandlerT(nil)
	capi.RefToCBrowserProcessHandlerT

	capi.RefToCClientT
	initial_url string
}

func (bph myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("L108:")
	capi.BrowserHostCreateBrowser(
		"Cefingo Example",
		bph.initial_url,
		bph.GetCClientT(),
	)
}

type myClient struct {
}

type myApp struct {
}

type myRenderProcessHander struct {
}

func (myRenderProcessHander) OnContextCreated(self *capi.CRenderProcessHandlerT,
	brower *capi.CBrowserT,
	frame *capi.CFrameT,
	context *capi.CV8contextT,
) {
	global := context.GetGlobal()

	my := capi.V8valueCreateObject(nil, nil)

	you := capi.V8valueCreateString("Wasm without Http Server")

	global.SetValueBykey("my", my)
	my.SetValueBykey("you", you)

	wasm, err := ioutil.ReadFile("wasm/test.wasm") // just pass the file name
	if err != nil {
		log.Panicln("L163:", err)
	}
	capi.Logf("L166: %d", len(wasm))
	v8wasm := capi.V8valueCreateArrayBuffer(wasm)
	capi.Logf("L168: %T, %v", v8wasm, unsafe.Pointer(v8wasm))

	my.SetValueBykey("wasm", v8wasm)

	capi.Logf("L156:")
}

func makeHtlmString() ([]byte, error) {
	wasmjs, err := ioutil.ReadFile(runtime.GOROOT() + "/misc/wasm/wasm_exec.js")
	if err != nil {
		return nil, err
	}
	temp, err := template.ParseFiles("html/wasm_exec_ab.html.template")
	if err != nil {
		return nil, err
	}
	params := struct {
		WasmJs template.JS
	}{
		template.JS(wasmjs),
	}
	out := bytes.Buffer{}
	err = temp.Execute(&out, params)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil

}
