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

var initial_url *string

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

	life_span_handler := myLifeSpanHandler{}
	capi.AllocCLifeSpanHandlerT().Bind(&life_span_handler)

	browser_process_handler := myBrowserProcessHandler{}
	capi.AllocCBrowserProcessHandlerT().Bind(&browser_process_handler)

	client := myClient{}
	capi.AllocCClient().Bind(&client)
	client.GetCClientT().AssocLifeSpanHandler(life_span_handler.GetCLifeSpanHandlerT())
	browser_process_handler.SetCClientT(client.GetCClientT())

	app := myApp{}
	capi.AllocCAppT().Bind(&app)
	app.GetCAppT().AssocBrowserProcessHandler(browser_process_handler.GetCBrowserProcessHandlerT())

	render_process_handler := myRenderProcessHander{}
	capi.AllocCRenderProcessHandlerT().Bind(&render_process_handler)
	app.GetCAppT().AssocRenderProcessHandler(render_process_handler.GetCRenderProcessHandlerT())

	capi.ExecuteProcess(app.GetCAppT())

	html, err := makeHtlmString()
	if err != nil {
		log.Panicln("Can not open html")
	}
	durl := dataurl.New(html, "text/html").String()
	initial_url = &durl

	s := capi.Settings{}
	s.LogSeverity = capi.LogSeverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	capi.Initialize(s, app.GetCAppT())

	capi.RunMessageLoop()
	defer capi.Shutdown()

}

type myLifeSpanHandler struct {
	capi.RefToCLifeSpanHandlerT
}

func (*myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.Logf("L89:")
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	capi.RefToCBrowserProcessHandlerT
	capi.RefToCClientT
}

func (bph *myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("L108:")
	capi.BrowserHostCreateBrowser(
		"Cefingo Example",
		*initial_url,
		bph.GetCClientT(),
	)
}

type myClient struct {
	capi.RefToCClientT
}

type myApp struct {
	capi.RefToCAppT
}

type myRenderProcessHander struct {
	capi.RefToCRenderProcessHandlerT
}

func (*myRenderProcessHander) OnContextCreated(self *capi.CRenderProcessHandlerT,
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
