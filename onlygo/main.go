package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/turutcrane/cefingo"
	"github.com/turutcrane/cefingo/v8"
)

var initial_url *string

const index_text = `
<html>
<head>
<link rel="stylesheet" type="text/css" href="/css/mystyle.css">
</head>
<body ID="body">
</body>
</html>
`

const inner_text = `
<div>
  <p>Hello Cefingo!!</p>
  <button id="B1">Button B1</button>
  <p id="P1"></p>
</div>
`

const css_text = `
body {
  font-size:30px;
}
#B1 {
  font-size: 20px;
  color: MediumSeaGreen;
}
`

func init() {
	prefix := fmt.Sprintf("[%d] ", os.Getpid())
	cefingo.Logger = log.New(os.Stdout, prefix, log.LstdFlags)
	// cefingo.RefCountLogOutput(true)

}

// var cefClient *cefingo.CClientT

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
	cBrowserProcessHandler := cefingo.AllocCBrowserProcessHandlerT(&browser_process_handler)

	app := myApp{}
	cefApp := cefingo.AllocCAppT(&app)
	cefingo.AssocBrowserProcessHandler(cefApp, cBrowserProcessHandler)

	render_process_handler := myRenderProcessHander{}
	cRenderProcessHandler := cefingo.AllocCRenderProcessHandlerT(&render_process_handler)
	cefApp.AssocRenderProcessHandler(cRenderProcessHandler)

	load_handler := myLoadHandler{}
	cLoadHander := cefingo.AllocCLoadHandlerT(&load_handler)
	cRenderProcessHandler.AssocLoadHandler(cLoadHander)

	cefingo.ExecuteProcess(cefApp)

	initial_url = flag.String("url", "cefingo.internal", "URL to Opne")
	flag.Parse()

	s := cefingo.Settings{}
	s.LogSeverity = cefingo.LogSeverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	// s.RemoteDebuggingPort = 8088
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
	factory := mySchemeHandlerFactory{}
	f := cefingo.AllocCSchemeHandlerFactoryT(&factory)
	cefingo.RegisterSchemeHandlerFactory("http", "cefingo.internal", f)

	life_span_handler := myLifeSpanHandler{}
	cLifeSpanHandler := cefingo.AllocCLifeSpanHandlerT(&life_span_handler)

	client := myClient{}
	cefClient := cefingo.AllocCClient(&client)
	cefClient.AssocLifeSpanHandler(cLifeSpanHandler)

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
	defer cefingo.BaseRelease(global)

	my := cefingo.V8valueCreateObject(nil, nil)
	defer cefingo.BaseRelease(my)

	msg := cefingo.V8valueCreateString("Cefingo Hello")
	defer cefingo.BaseRelease(msg)

	global.SetValueBykey("my", my)
	my.SetValueBykey("msg", msg)

}

type mySchemeHandlerFactory struct{}

func (factory *mySchemeHandlerFactory) Create(
	self *cefingo.CSchemeHandlerFactoryT,
	browser *cefingo.CBrowserT,
	frame *cefingo.CFrameT,
	scheme_name string,
	request *cefingo.CRequestT,
) (handler *cefingo.CResourceHandlerT) {
	url, err := url.Parse(request.GetUrl())
	if err != nil {
		return nil
	}

	cefingo.Logf("L329: %s, %s", url, url.Hostname())

	if url.Hostname() == "cefingo.internal" {
		rh := myResourceHandler{r: request}
		switch url.Path {
		case "/":
			rh.mime = "text/html"
			rh.text = index_text
		case "/css/mystyle.css":
			rh.mime = "text/css"
			rh.text = css_text
		}
		cefingo.BaseAddRef(request)
		handler = cefingo.AllocCResourceHanderT(&rh)
	}
	return handler
}

type myResourceHandler struct {
	cefingo.DefaultResourceHandler
	r    *cefingo.CRequestT
	text string
	mime string
}

func (rh *myResourceHandler) ProcessRequest(
	self *cefingo.CResourceHandlerT,
	request *cefingo.CRequestT,
	callback *cefingo.CCallbackT,
) bool {
	rh.r = request
	cefingo.Logf("L339: %s", request.GetUrl())
	callback.Cont()
	return true
}

func (rh *myResourceHandler) GetResponseHeaders(
	self *cefingo.CResourceHandlerT,
	response *cefingo.CResponseT,
	response_length *int64,
	redirectUrl *string,
) {
	u, err := url.Parse(rh.r.GetUrl())
	if err != nil {
		cefingo.Panicf("L393: Error")
	}
	cefingo.Logf("L391: %s", u.Path)
	response.SetMimeType(rh.mime)
	h := []cefingo.StringMap{
		{Key: "Content-Type", Value: rh.mime + "; charset=utf-8"},
	}
	response.SetStatus(200)
	// response.SetStatusText("OK")
	response.SetHeaderMap(h)

	*response_length = int64(len(rh.text))
}

func (rh *myResourceHandler) ReadResponse(
	self *cefingo.CResourceHandlerT,
	data_out []byte,
	bytes_to_read int,
	bytes_read *int,
	callback *cefingo.CCallbackT,
) bool {
	l := len(rh.text)
	buf := []byte(rh.text)
	l = min(l, len(buf))
	for i, b := range buf[:l] {
		data_out[i] = b
	}
	*bytes_read = l
	cefingo.Logf("L409: %d, %d", len(rh.text), l)
	return true
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type myLoadHandler struct {
	cefingo.DefaultLoadHandler
}

func (*myLoadHandler) OnLoadEnd(
	self *cefingo.CLoadHandlerT,
	browser *cefingo.CBrowserT,
	frame *cefingo.CFrameT,
	httpStatusCode int,
) {
	context := frame.GetV8context()
	defer cefingo.BaseRelease(context)

	if context.Enter() {
		defer context.Exit()

		c := v8.GetContext()
		defer v8.ReleaseContext(c)
		cefingo.Logf("L284: is_same:%t", context.IsSame(c.V8context))

		v, err := c.GetElementById("body")
		if err == nil {
			defer v.Release()
			cefingo.Logf("L289: %v", v.HasValueBykey("innerHTML"))
			html := v8.CreateString(inner_text)
			v.SetValueBykey("innerHTML", html)
		}
		v, err = c.GetElementById("B1")
		if err != nil {
			cefingo.Logf("L300: %v", err)
		} else {
			defer v.Release()
			v.AddEventListener(v8.EventClick, func(*cefingo.CV8valueT) error {
				c1 := v8.GetContext()
				defer v8.ReleaseContext(c1)
				// _, err := c1.EvalString("alert('B1 Clicked: ' + my.msg);")
				c1.Alertf("B1 Clicked: %t", c1.V8context.IsSame(c.V8context))
				return err
			})
		}
	} else {
		cefingo.Logf("L293: Did not enterd in V8 Context")
	}
}
