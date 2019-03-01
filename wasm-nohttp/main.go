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

	"github.com/turutcrane/cefingo"
	"github.com/vincent-petithory/dataurl"
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

	render_process_handler := myRenderProcessHander{}
	cRenderProcessHandler := cefingo.AllocCRenderProcessHandlerT(&render_process_handler)
	cefApp.AssocRenderProcessHandler(cRenderProcessHandler)

	cefingo.ExecuteProcess(cefApp)

	html, err := makeHtlmString()
	if err != nil {
		log.Panicln("Can not open html")
	}
	durl := dataurl.New(html, "text/html").String()
	initial_url = &durl

	s := cefingo.Settings{}
	s.LogSeverity = cefingo.LogSeverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	cefingo.Initialize(s, cefApp)

	cefingo.RunMessageLoop()
	defer cefingo.Shutdown()

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

type myRenderProcessHander struct {
	cefingo.DefaultRenderProcessHander
}

func (*myRenderProcessHander) OnContextCreated(self *cefingo.CRenderProcessHandlerT,
	brower *cefingo.CBrowserT,
	frame *cefingo.CFrameT,
	context *cefingo.CV8contextT,
) {
	global := context.GetGlobal()

	my := cefingo.V8valueCreateObject(nil, nil)

	you := cefingo.V8valueCreateString("Wasm without Http Server")

	global.SetValueBykey("my", my)
	my.SetValueBykey("you", you)

	wasm, err := ioutil.ReadFile("wasm/test.wasm") // just pass the file name
	if err != nil {
		log.Panicln("L163:", err)
	}
	cefingo.Logf("L166: %d", len(wasm))
	v8wasm := cefingo.V8valueCreateArrayBuffer(wasm)
	cefingo.Logf("L168: %T, %v", v8wasm, unsafe.Pointer(v8wasm))

	my.SetValueBykey("wasm", v8wasm)

	cefingo.Logf("L156:")
}

func makeHtlmString() ([]byte, error) {
	wasmjs, err := ioutil.ReadFile("html/wasm_exec.js")
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
