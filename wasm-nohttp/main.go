package main

import (
	"bytes"
	"embed"
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

	mainArgs := capi.NewCMainArgsT()
	cef.CMainArgsTSetInstance(mainArgs)


	client := &myClient{}
	capi.AllocCClientT().Bind(client)
	defer client.GetCClientT().UnbindAll()

	capi.AllocCLifeSpanHandlerT().Bind(client)
	defer client.GetCClientT().UnbindAll()

	app := &myApp{}
	capi.AllocCAppT().Bind(app)
	defer app.GetCAppT().UnbindAll()

	capi.AllocCBrowserProcessHandlerT().Bind(app)
	defer app.GetCBrowserProcessHandlerT().UnbindAll()
	
	app.SetCClientT(client.GetCClientT())

	capi.AllocCRenderProcessHandlerT().Bind(app)
	defer app.GetCRenderProcessHandlerT().UnbindAll()

	cef.ExecuteProcess(mainArgs, app.GetCAppT())

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
	cef.Initialize(mainArgs, s, app.GetCAppT())

	capi.RunMessageLoop()
	defer capi.Shutdown()

}

type myLifeSpanHandler struct {
	capi.RefToCLifeSpanHandlerT
}

func init() {
	var _ capi.OnBeforeCloseHandler = (*myLifeSpanHandler)(nil)
}

func (*myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.Logf("L89:")
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	// this reference forms an UNgabagecollectable circular reference
	// To GC, call myBrowserProcessHandler.GetCBrowserProcessHandlerT().Unbind()
	capi.RefToCBrowserProcessHandlerT

	capi.RefToCClientT
	initial_url string
}

func init() {
	var _ capi.OnContextInitializedHandler = (*myBrowserProcessHandler)(nil)
}

func (bph *myBrowserProcessHandler) GetBrowserProcessHandler(*capi.CAppT) *capi.CBrowserProcessHandlerT {
	return bph.GetCBrowserProcessHandlerT()
}

func (bph *myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("L108:")
	windowInfo := capi.NewCWindowInfoT()
	windowInfo.SetStyle(win32api.WsOverlappedwindow | win32api.WsClipchildren |
		win32api.WsClipsiblings | win32api.WsVisible)
	windowInfo.SetParentWindow(nil)
	windowInfo.SetX(win32api.CwUsedefault)
	windowInfo.SetY(win32api.CwUsedefault)
	windowInfo.SetWidth(win32api.CwUsedefault)
	windowInfo.SetHeight(win32api.CwUsedefault)
	windowInfo.SetWindowName("Cefingo Wasm No-http Example")

	browserSettings := capi.NewCBrowserSettingsT()

	capi.BrowserHostCreateBrowser(windowInfo,
		bph.GetCClientT(),
		bph.initial_url,
		browserSettings,
		nil, nil,
	)
}

type myClient struct {
	capi.RefToCClientT
	myLifeSpanHandler
}

func init() {
	var _ capi.GetLifeSpanHandlerHandler = (*myClient)(nil)
}

func (lsh *myLifeSpanHandler) GetLifeSpanHandler(*capi.CClientT) *capi.CLifeSpanHandlerT {
	return lsh.GetCLifeSpanHandlerT()
}

type myApp struct {
	capi.RefToCAppT

	myBrowserProcessHandler
	myRenderProcessHandler
}

func init() {
	var _ capi.GetBrowserProcessHandlerHandler = (*myApp)(nil)
	var _ capi.GetRenderProcessHandlerHandler = (*myApp)(nil)
}

type myRenderProcessHandler struct {
	capi.RefToCRenderProcessHandlerT
}

func init() {
	var _ capi.OnContextCreatedHandler = (*myRenderProcessHandler)(nil)
}

func (rph *myRenderProcessHandler) GetRenderProcessHandler(*capi.CAppT) *capi.CRenderProcessHandlerT {
	capi.Logf("T173:")
	return rph.GetCRenderProcessHandlerT()
}

func (*myRenderProcessHandler) OnContextCreated(self *capi.CRenderProcessHandlerT,
	brower *capi.CBrowserT,
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
