package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/turutcrane/cefingo/capi"
	"github.com/turutcrane/cefingo/cef"
	"github.com/turutcrane/cefingo/v8api"
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

	browser_process_handler := myBrowserProcessHandler{}
	capi.AllocCBrowserProcessHandlerT().Bind(&browser_process_handler)
	defer browser_process_handler.SetCBrowserProcessHandlerT(nil)

	app := capi.AllocCAppT().Bind(&myApp{})
	app.AssocBrowserProcessHandlerT(browser_process_handler.GetCBrowserProcessHandlerT())

	render_process_handler :=
		capi.AllocCRenderProcessHandlerT().Bind(&myRenderProcessHander{})
	app.AssocRenderProcessHandlerT(render_process_handler)

	load_handler :=
		capi.AllocCLoadHandlerT().Bind(&myLoadHandler{})
	render_process_handler.AssocLoadHandlerT(load_handler)

	capi.ExecuteProcess(app)

	initial_url := flag.String("url", internalHostName, "URL to Opne")
	flag.Parse()

	browser_process_handler.initial_url = *initial_url

	s := capi.Settings{}
	s.LogSeverity = capi.LogseverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	// s.RemoteDebuggingPort = 8088
	capi.Initialize(s, app)

	capi.RunMessageLoop()
	defer capi.Shutdown()

}

type myLifeSpanHandler struct {
}
func init() {
	var _ capi.OnBeforeCloseHandler = myLifeSpanHandler{}
}

func (myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.Logf("L89:")
	capi.QuitMessageLoop()
}

type myBrowserProcessHandler struct {
	// this reference forms an UNgabagecollectable circular reference
	// To GC, call myBrowserProcessHandler.SetCBrowserProcessHandlerT(nil)
	capi.RefToCBrowserProcessHandlerT

	initial_url string
}
func init() {
	var _ capi.OnContextInitializedHandler = myBrowserProcessHandler{}
}

func (bph myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("L108:")
	factory := capi.AllocCSchemeHandlerFactoryT().Bind(&mySchemeHandlerFactory{})
	capi.RegisterSchemeHandlerFactory(
		"http",
		internalHostName,
		factory,
	)

	life_span_handler := capi.AllocCLifeSpanHandlerT().Bind(&myLifeSpanHandler{})

	client := capi.AllocCClientT().Bind(&myClient{})
	client.AssocLifeSpanHandlerT(life_span_handler)

	capi.BrowserHostCreateBrowser(
		"Cefingo Example",
		bph.initial_url,
		client,
	)
}

type myClient struct {
}

type myApp struct {
}

type myRenderProcessHander struct {
}

func init() {
	var _ capi.OnContextCreatedHandler = myRenderProcessHander{}
}

func (myRenderProcessHander) OnContextCreated(self *capi.CRenderProcessHandlerT,
	brower *capi.CBrowserT,
	frame *capi.CFrameT,
	context *capi.CV8contextT,
) {
	global := context.GetGlobal()

	my := capi.V8valueCreateObject(nil, nil)

	msg := capi.V8valueCreateString("Cefingo Hello")

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
	var _ capi.CreateHandler = mySchemeHandlerFactory{}
}

func (factory mySchemeHandlerFactory) Create(
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
		rh.SetCRequestT(request)
		switch url.Path {
		case "/":
			rh.mime = "text/html"
			rh.text = index_text
		case "/css/mystyle.css":
			rh.mime = "text/css"
			rh.text = css_text
		}
		handler = capi.AllocCResourceHandlerT().Bind(&rh)
	}
	return handler
}

type myResourceHandler struct {
	capi.RefToCRequestT
	text string
	mime string
}

func init() {
	var _ capi.ProcessRequestHandler = myResourceHandler{}
	var _ capi.GetResponseHeadersHandler = myResourceHandler{}
	var _ capi.ReadResponseHandler = myResourceHandler{}
}

func (rh myResourceHandler) ProcessRequest(
	self *capi.CResourceHandlerT,
	request *capi.CRequestT,
	callback *capi.CCallbackT,
) bool {
	rh.SetCRequestT(request)
	capi.Logf("L339: %s", request.GetUrl())
	callback.Cont()
	return true
}

func (rh myResourceHandler) GetResponseHeaders(
	self *capi.CResourceHandlerT,
	response *capi.CResponseT,
	response_length *int64,
	redirectUrl *string,
) {
	u, err := url.Parse(rh.GetCRequestT().GetUrl())
	if err != nil {
		capi.Panicf("L393: Error")
	}
	capi.Logf("L391: %s", u.Path)
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

	*response_length = int64(len(rh.text))
}

func (rh myResourceHandler) ReadResponse(
	self *capi.CResourceHandlerT,
	data_out []byte,
	bytes_read *int,
	callback *capi.CCallbackT,
) bool {
	l := len(rh.text)
	buf := []byte(rh.text)
	l = min(l, len(data_out))
	for i, b := range buf[:l] {
		data_out[i] = b
	}
	*bytes_read = l
	capi.Logf("L409: %d, %d", len(rh.text), l)
	return true
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type myLoadHandler struct {
}

func init() {
	var _ capi.OnLoadEndHandler = myLoadHandler{}
}

func (myLoadHandler) OnLoadEnd(
	self *capi.CLoadHandlerT,
	browser *capi.CBrowserT,
	frame *capi.CFrameT,
	httpStatusCode int,
) {
	context := frame.GetV8context()

	if context.Enter() {
		defer context.Exit()

		c, err := v8.GetContext()
		if err != nil {
			capi.Logf("280: get context; %+v", err)
			return
		}
		capi.Logf("L284: is_same:%t", context.IsSame(c.V8context))

		b1, err := c.GetElementById("B1")
		if err == nil {
			b1.AddEventListener(v8.EventClick, v8.EventHandlerFunc(func(object v8.Value, event v8.Value) error {
				c1, err := v8.GetContext()
				if err != nil {
					return errors.Wrap(err, "get context")
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
					c2, err := v8.GetContext()
					if err != nil {
						return errors.Wrap(err, "E311: get context")
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
