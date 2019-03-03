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
	"github.com/turutcrane/cefingo/v8"
)

var initial_url *string

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

// var cefClient *capi.CClientT

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
	cBrowserProcessHandler := capi.AllocCBrowserProcessHandlerT(&browser_process_handler)

	app := myApp{}
	cefApp := capi.AllocCAppT(&app)
	cefApp.AssocBrowserProcessHandler(cBrowserProcessHandler)

	render_process_handler := myRenderProcessHander{}
	cRenderProcessHandler := capi.AllocCRenderProcessHandlerT(&render_process_handler)
	cefApp.AssocRenderProcessHandler(cRenderProcessHandler)

	load_handler := myLoadHandler{}
	cLoadHander := capi.AllocCLoadHandlerT(&load_handler)
	cRenderProcessHandler.AssocLoadHandler(cLoadHander)

	capi.ExecuteProcess(cefApp)

	initial_url = flag.String("url", internalHostName, "URL to Opne")
	flag.Parse()

	s := capi.Settings{}
	s.LogSeverity = capi.LogSeverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	// s.RemoteDebuggingPort = 8088
	capi.Initialize(s, cefApp)

	capi.RunMessageLoop()
	defer capi.Shutdown()

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
	factory := mySchemeHandlerFactory{}
	f := capi.AllocCSchemeHandlerFactoryT(&factory)
	capi.RegisterSchemeHandlerFactory("http", internalHostName, f)

	life_span_handler := myLifeSpanHandler{}
	cLifeSpanHandler := capi.AllocCLifeSpanHandlerT(&life_span_handler)

	client := myClient{}
	cefClient := capi.AllocCClient(&client)
	cefClient.AssocLifeSpanHandler(cLifeSpanHandler)

	capi.BrowserHostCreateBrowser("Cefingo Example", *initial_url, cefClient)
}

type myClient struct {
	capi.DefaultClient
}

type myApp struct {
	capi.DefaultApp
}

type myRenderProcessHander struct {
	capi.DefaultRenderProcessHander
}

func (*myRenderProcessHander) OnContextCreated(self *capi.CRenderProcessHandlerT,
	brower *capi.CBrowserT,
	frame *capi.CFrameT,
	context *capi.CV8contextT,
) {
	global := context.GetGlobal()

	my := capi.V8valueCreateObject(nil, nil)

	msg := capi.V8valueCreateString("Cefingo Hello")

	global.SetValueBykey("my", my)
	my.SetValueBykey("msg", msg)

}

type mySchemeHandlerFactory struct{}
const internalHostName = "capi.internal"

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
		rh := myResourceHandler{r: request}
		switch url.Path {
		case "/":
			rh.mime = "text/html"
			rh.text = index_text
		case "/css/mystyle.css":
			rh.mime = "text/css"
			rh.text = css_text
		}
		capi.BaseAddRef(request)
		handler = capi.AllocCResourceHanderT(&rh)
	}
	return handler
}

type myResourceHandler struct {
	capi.DefaultResourceHandler
	r    *capi.CRequestT
	text string
	mime string
}

func (rh *myResourceHandler) ProcessRequest(
	self *capi.CResourceHandlerT,
	request *capi.CRequestT,
	callback *capi.CCallbackT,
) bool {
	rh.r = request
	capi.Logf("L339: %s", request.GetUrl())
	callback.Cont()
	return true
}

func (rh *myResourceHandler) GetResponseHeaders(
	self *capi.CResourceHandlerT,
	response *capi.CResponseT,
	response_length *int64,
	redirectUrl *string,
) {
	u, err := url.Parse(rh.r.GetUrl())
	if err != nil {
		capi.Panicf("L393: Error")
	}
	capi.Logf("L391: %s", u.Path)
	response.SetMimeType(rh.mime)
	h := []capi.StringMap{
		{Key: "Content-Type", Value: rh.mime + "; charset=utf-8"},
	}
	response.SetStatus(200)
	// response.SetStatusText("OK")
	response.SetHeaderMap(h)

	*response_length = int64(len(rh.text))
}

func (rh *myResourceHandler) ReadResponse(
	self *capi.CResourceHandlerT,
	data_out []byte,
	bytes_to_read int,
	bytes_read *int,
	callback *capi.CCallbackT,
) bool {
	l := len(rh.text)
	buf := []byte(rh.text)
	l = min(l, len(buf))
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
	capi.DefaultLoadHandler
}

func (*myLoadHandler) OnLoadEnd(
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
