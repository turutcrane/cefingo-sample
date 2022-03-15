package main

import (
	"bytes"
	"embed"
	"fmt"
	"runtime"

	// "fmt"
	"html/template"
	"io"
	"log"
	"os"
	"time"
	"unsafe"

	"github.com/turutcrane/cefingo/capi"
	"github.com/turutcrane/cefingo/cef"
	"github.com/turutcrane/win32api"
	"github.com/vincent-petithory/dataurl"
)

//go:embed html
var htmlFs embed.FS

//go:embed wasm/test.wasm
var testWasm []byte

func init() {
	// capi.Initialize(i.e. cef_initialize) and some function should be called on
	// the main application thread to initialize the CEF browser process
	runtime.LockOSThread()
	prefix := fmt.Sprintf("[%d] ", os.Getpid())
	capi.Logger = log.New(os.Stdout, prefix, log.LstdFlags)
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

	mainArgs := capi.NewCMainArgsT()
	cef.CMainArgsTSetInstance(mainArgs)

	doCef(mainArgs)
	runtime.GC()

	capi.Shutdown()
}

func doCef(mainArgs *capi.CMainArgsT) {
	client := &myClient{}
	client.client = capi.NewCClientT(client)
	defer client.client.Unref() // .UnbindAll()

	client.lifeSpanHandler = capi.NewCLifeSpanHandlerT(client)
	defer client.lifeSpanHandler.Unref() // .UnbindAll()

	app := &myApp{}
	app.app = capi.NewCAppT(app)
	defer app.app.Unref() // .UnbindAll()

	app.browserProcessHandler = capi.NewCBrowserProcessHandlerT(app)
	defer app.browserProcessHandler.Unref() // .UnbindAll()
	
	app.myBrowserProcessHandler.client = client

	app.renderProcessHandler= capi.NewCRenderProcessHandlerT(app)
	defer app.renderProcessHandler.Unref() //.UnbindAll()

	cef.ExecuteProcess(mainArgs, app.app)

	html, err := makeHtlmString()
	if err != nil {
		log.Panicln("Can not open html")
	}
	durl := dataurl.New(html, "text/html").String()
	app.initial_url = durl

	s := capi.NewCSettingsT()
	s.SetLogSeverity(capi.LogseverityWarning)
	s.SetNoSandbox(false)
	s.SetMultiThreadedMessageLoop(false)
	s.SetRemoteDebuggingPort(8088)
	cef.Initialize(mainArgs, s, app.app)

	capi.RunMessageLoop()

}

type myLifeSpanHandler struct {
	lifeSpanHandler *capi.CLifeSpanHandlerT
}

func init() {
	var _ capi.OnBeforeCloseHandler = (*myLifeSpanHandler)(nil)
}

func (*myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, browser *capi.CBrowserT) {
	capi.Logf("L89:")
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	browserProcessHandler *capi.CBrowserProcessHandlerT

	client *myClient
	initial_url string
}

func init() {
	var _ capi.OnContextInitializedHandler = (*myBrowserProcessHandler)(nil)
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
	windowInfo.SetWindowName("Cefingo Wasm No-http Example")

	browserSettings := capi.NewCBrowserSettingsT()

	capi.BrowserHostCreateBrowser(windowInfo,
		bph.client.client,
		bph.initial_url,
		browserSettings,
		nil, nil,
	)
}

type myClient struct {
	client *capi.CClientT
	myLifeSpanHandler
}

func init() {
	var _ capi.GetLifeSpanHandlerHandler = (*myClient)(nil)
}

func (lsh *myLifeSpanHandler) GetLifeSpanHandler(*capi.CClientT) *capi.CLifeSpanHandlerT {
	return lsh.lifeSpanHandler
}

type myApp struct {
	app *capi.CAppT

	myBrowserProcessHandler
	myRenderProcessHandler
}

func init() {
	var _ capi.GetBrowserProcessHandlerHandler = (*myApp)(nil)
	var _ capi.GetRenderProcessHandlerHandler = (*myApp)(nil)
}

type myRenderProcessHandler struct {
	renderProcessHandler *capi.CRenderProcessHandlerT
}

func init() {
	var _ capi.OnContextCreatedHandler = (*myRenderProcessHandler)(nil)
}

func (rph *myRenderProcessHandler) GetRenderProcessHandler(*capi.CAppT) *capi.CRenderProcessHandlerT {
	capi.Logf("T173:")
	return rph.renderProcessHandler
}

func (*myRenderProcessHandler) OnContextCreated(self *capi.CRenderProcessHandlerT,
	browser *capi.CBrowserT,
	frame *capi.CFrameT,
	context *capi.CV8contextT,
) {
	global := context.GetGlobal()

	my := capi.V8valueCreateObject(nil, nil)

	you := capi.V8valueCreateString("Wasm without Http Server")

	if ok := global.SetValueBykey("my", my, capi.V8PropertyAttributeNone); !ok {
		capi.Logf("T124: can not set my")
	}
	if ok := my.SetValueBykey("you", you, capi.V8PropertyAttributeNone); !ok {
		capi.Logf("T127: can not set you")
	}

	// wasm, err := ioutil.ReadFile("wasm/test.wasm") // just pass the file name
	// if err != nil {
	// 	log.Panicln("L163:", err)
	// }
	capi.Logf("L166: %d", len(testWasm))
	v8wasm := capi.CreateArrayBuffer(testWasm)
	defer v8wasm.Unref()
	capi.Logf("L168: %T, %v", v8wasm, unsafe.Pointer(v8wasm))

	if ok := my.SetValueBykey("wasm", v8wasm, capi.V8PropertyAttributeNone); !ok {
		capi.Logf("T139: can not set you")
	}

	capi.Logf("L156:")
}

func makeHtlmString() ([]byte, error) {
	r, err := os.Open(runtime.GOROOT() + "/misc/wasm/wasm_exec.js")
	if err != nil {
		return nil, err
	}
	wasmjs, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	temp, err := template.ParseFS(htmlFs, "html/wasm_exec_ab.html.template")
	// temp, err := template.ParseFiles("html/wasm_exec_ab.html.template")
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
