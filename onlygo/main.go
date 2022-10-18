package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/turutcrane/cefingo/capi"
	"github.com/turutcrane/cefingo/cef"
	"github.com/turutcrane/cefingo/v8api"
	"github.com/turutcrane/win32api"
)

const index_text = `
<html>
<head>
<link rel="stylesheet" type="text/css" href="/css/mystyle.css">
</head>
<body ID="body">
<div>
  <p>Hello Cefingo!!</p>
  <button id="B1">Button B1</button>
  <br><br>
  <button id="B2">Button B2</button>
  <div id="DIV1"></div>
</div>
</body>
</html>
`
const css_text = `
body {
  font-size:30px;
}
button {
  font-size: 20px;
  color: MediumSeaGreen;
}
#B2 {
  color: FireBrick;
}
`

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

	app := &myApp{}
	app.app = capi.NewCAppT(app)
	defer app.app.Unref() // .UnbindAll()

	app.browserProcessHandler =  capi.NewCBrowserProcessHandlerT(app)
	defer app.browserProcessHandler.Unref() // .UnbindAll()

	app.renderProcessHandler =  capi.NewCRenderProcessHandlerT(app)
	defer app.renderProcessHandler.Unref() // .UnbindAll()

	app.loadHandler =  capi.NewCLoadHandlerT(app)
	defer app.loadHandler.Unref() // .UnbindAll()

	cef.ExecuteProcess(mainArgs, app.app)

	initial_url := flag.String("url", internalHostName, "URL to Opne")
	flag.Parse()

	app.initial_url = *initial_url

	s := capi.NewCSettingsT()
	s.SetLogSeverity(capi.LogseverityWarning)
	s.SetNoSandbox(false)
	s.SetMultiThreadedMessageLoop(false)
	s.SetRemoteDebuggingPort(8088)

	cef.Initialize(mainArgs, s, app.app)

	capi.RunMessageLoop()
	defer capi.Shutdown()

}

type myLifeSpanHandler struct {
	lifeSpanHandler *capi.CLifeSpanHandlerT
}

func init() {
	var _ capi.OnBeforeCloseHandler = (*myLifeSpanHandler)(nil)
}

func (lsh *myLifeSpanHandler) GetLifeSpanHandler(*capi.CClientT) *capi.CLifeSpanHandlerT {
	return lsh.lifeSpanHandler
}

func (lsh *myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, browser *capi.CBrowserT) {
	capi.QuitMessageLoop()
	if client, ok := self.Handler().(*myClient); ok {
		capi.Logf("L124:")
		client.client.Unref() // .UnbindAll()
	}
	lsh.lifeSpanHandler.Unref() // .UnbindAll()
}

type myBrowserProcessHandler struct {
	// this reference forms an UNgabagecollectable circular reference
	// To GC, call myBrowserProcessHandler.GetCBrowserProcessHandlerT().UnbindAll()
	browserProcessHandler *capi.CBrowserProcessHandlerT

	initial_url string
}

func init() {
	var _ capi.OnContextInitializedHandler = (*myBrowserProcessHandler)(nil)
}

func (bph *myBrowserProcessHandler) GetBrowserProcessHandler(*capi.CAppT) *capi.CBrowserProcessHandlerT {
	return bph.browserProcessHandler
}

func (bph *myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	factory := capi.NewCSchemeHandlerFactoryT(&mySchemeHandlerFactory{})
	defer factory.Unref()
	capi.RegisterSchemeHandlerFactory(
		"http",
		internalHostName,
		factory,
	)

	client := &myClient{}
	client.client = capi.NewCClientT(client)
	client.lifeSpanHandler = capi.NewCLifeSpanHandlerT(client)

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
	windowInfo.SetWindowName("Cefingo Only Go Example")

	browserSettings := capi.NewCBrowserSettingsT()

	capi.BrowserHostCreateBrowser(windowInfo,
		client.client,
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

type myApp struct {
	app *capi.CAppT

	myBrowserProcessHandler
	myRenderProcessHander
}

func init() {
	var _ capi.GetBrowserProcessHandlerHandler = (*myApp)(nil)
	var _ capi.GetRenderProcessHandlerHandler = (*myApp)(nil)
}

type myRenderProcessHander struct {
	renderProcessHandler *capi.CRenderProcessHandlerT
	myLoadHandler
}

func init() {
	var _ capi.OnContextCreatedHandler = (*myRenderProcessHander)(nil)
	var _ capi.CRenderProcessHandlerTGetLoadHandlerHandler = (*myRenderProcessHander)(nil)
}

func (rph *myRenderProcessHander) GetRenderProcessHandler(*capi.CAppT) *capi.CRenderProcessHandlerT {
	return rph.renderProcessHandler
}

func (*myRenderProcessHander) OnContextCreated(self *capi.CRenderProcessHandlerT,
	browser *capi.CBrowserT,
	frame *capi.CFrameT,
	context *capi.CV8contextT,
) {
	global := context.GetGlobal()

	my := capi.V8valueCreateObject(nil, nil)
	defer my.Unref()
	msg := capi.V8valueCreateString("Cefingo Hello")
	defer msg.Unref()

	if ok := global.SetValueBykey("my", my, capi.V8PropertyAttributeNone); !ok {
		capi.Logf("T163: can not set my")
	}
	if ok := my.SetValueBykey("msg", msg, capi.V8PropertyAttributeNone); !ok {
		capi.Logf("T163: can not set msg")
	}

}

type mySchemeHandlerFactory struct {
}

const internalHostName = "capi.internal"

func init() {
	var _ capi.CreateHandler = (*mySchemeHandlerFactory)(nil)
}

func (factory *mySchemeHandlerFactory) Create(
	self *capi.CSchemeHandlerFactoryT,
	browser *capi.CBrowserT,
	frame *capi.CFrameT,
	scheme_name string,
	request *capi.CRequestT,
) (handler *capi.CResourceHandlerT) {
	url, err := url.Parse(request.GetUrl())
	if err != nil {
		return nil
	}

	capi.Logf("L329: %s, %s", url, url.Hostname())

	if url.Hostname() == internalHostName {
		rh := myResourceHandler{}
		rh.requestUrl = url
		switch url.Path {
		case "/":
			rh.mime = "text/html"
			rh.text = []byte(index_text)
		case "/css/mystyle.css":
			rh.mime = "text/css"
			rh.text = []byte(css_text)
		}
		handler = capi.NewCResourceHandlerT(&rh)
	}
	return handler.Pass()
}

type myResourceHandler struct {
	// capi.RefToCRequestT
	requestUrl *url.URL
	text       []byte
	mime       string
	next       int
}

func init() {
	var _ capi.ProcessRequestHandler = (*myResourceHandler)(nil)
	var _ capi.GetResponseHeadersHandler = (*myResourceHandler)(nil)
	var _ capi.CResourceHandlerTReadHandler = (*myResourceHandler)(nil)
}

func (rh *myResourceHandler) ProcessRequest(
	self *capi.CResourceHandlerT,
	request *capi.CRequestT,
	callback *capi.CCallbackT,
) bool {
	u , err := url.Parse(request.GetUrl())
	if err != nil {
		capi.Panicf("L393: Error")
	}
	rh.requestUrl = u
	capi.Logf("L339: %s", request.GetUrl())
	callback.Cont()
	return true
}

func (rh *myResourceHandler) GetResponseHeaders(
	self *capi.CResourceHandlerT,
	response *capi.CResponseT,
) (int64, string) {

	capi.Logf("L391: %s", rh.requestUrl.Path)
	response.SetMimeType(rh.mime)
	// h := []capi.StringMap{
	// 	{Key: "Content-Type", Value: rh.mime + "; charset=utf-8"},
	// }
	response.SetStatus(200)
	// response.SetStatusText("OK")
	h := cef.NewStringMultimap()
	capi.StringMultimapAppend(h.CefObject(), "Content-Type", rh.mime+"; charset=utf-8")
	response.SetHeaderMap(h.CefObject())
	// response.SetHeaderMap(h)

	return int64(len(rh.text)), ""
}

// ReadResponse method is deprecated from cef 75
func (rh *myResourceHandler) Read(
	self *capi.CResourceHandlerT,
	data_out []byte,
	callback *capi.CResourceReadCallbackT,
) (bool, int) {
	l := min(len(data_out), len(rh.text)-rh.next)
	for i := 0; i < l; i++ {
		data_out[i] = rh.text[rh.next+i]
	}

	rh.next = rh.next + l
	capi.Logf("L409: %d, %d, %d", len(rh.text), l, rh.next)
	ret := true
	if l <= 0 {
		ret = false
	}
	return ret, l
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type myLoadHandler struct {
	loadHandler *capi.CLoadHandlerT
}

func init() {
	var _ capi.OnLoadEndHandler = (*myLoadHandler)(nil)
}

func (lh *myLoadHandler) GetLoadHandler(*capi.CRenderProcessHandlerT) *capi.CLoadHandlerT {
	return lh.loadHandler
}

func (*myLoadHandler) OnLoadEnd(
	self *capi.CLoadHandlerT,
	browser *capi.CBrowserT,
	frame *capi.CFrameT,
	httpStatusCode int,
) {
	context := frame.GetV8context()
	defer context.Unref()

	if context.Enter() {
		defer context.Exit()

		c, err := v8.GetCurrentContext()
		if err != nil {
			capi.Logf("280: get context; %+v", err)
			return
		}
		capi.Logf("L284: is_same:%t", context.IsSame(c.V8context))

		b1, err := c.GetElementById("B1")
		if err == nil {
			b1.AddEventListener(v8.EventClick, v8.EventHandlerFunc(func(object v8.Value, event v8.Value) error {
				c1, err := v8.GetCurrentContext()
				if err != nil {
					return fmt.Errorf("E386: get context: %w", err)
				}
				// _, err := c1.Eval("alert('B1 Clicked: ' + my.msg);")
				c1.Alertf("B1 Clicked !!: %s", time.Now().Format("03:04:05"))
				return nil
			}))
		} else {
			capi.Logf("L300: %v", err)
		}

		b2, err := c.GetElementById("B2")
		if err == nil {
			b2.AddEventListener(v8.EventClick, v8.EventHandlerFunc(
				func(object v8.Value, event v8.Value) error {
					c2, err := v8.GetCurrentContext()
					if err != nil {
						return fmt.Errorf("E311: get context: %w", err)
					}
					p1, err := c2.GetElementById("DIV1")
					if err == nil {
						html := v8.CreateString(fmt.Sprintf("<p>Hello, Umeda-Go! %s</p>", time.Now().Format("03:04:05 MST")))
						p1.SetValueBykey("innerHTML", html)
					}
					return err
				}))
		} else {
			capi.Logf("L302: Did not hab #B2 element.: %v", err)
		}
	}
}

// Example of get a string value of js variable
//   <script>
//	var cef = {};
//	cef.msg = "A message";
//   </script>
// func get_cef_msg(c *v8.Context) string {
// 	cef := c.Global.GetValueBykey("cef")
// 	defer capi.BaseRelease(cef)
// 	msg := cef.GetValueBykey("msg")
// 	defer capi.BaseRelease(msg)
//
// 	s := msg.GetString()
// 	return s
// }
