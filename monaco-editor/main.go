package main

import (
	"embed"
	"log"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	e "github.com/julvo/htmlgo"
	a "github.com/julvo/htmlgo/attributes"
	"github.com/turutcrane/cefingo/capi"
	"github.com/turutcrane/cefingo/cef"
	"github.com/turutcrane/cefingo/v8api"
	"github.com/turutcrane/win32api"
)

//go:embed package
var monacoPkg embed.FS

func init() {
	// capi.Initialize(i.e. cef_initialize) and some function should be called on
	// the main application thread to initialize the CEF browser process
	runtime.LockOSThread()

	// prefix := fmt.Sprintf("[%d] ", os.Getpid())
	// capi.Logger = log.New(os.Stdout, prefix, log.LstdFlags)
	// capi.RefCountLogOutput(true)
	// capi.RefCountLogTrace(true)
}

func main() {
	// defer log.Println("T19: Graceful Shutdowned")
	// log.Println("T20: started:", "Pid:", os.Getpid(), "PPid:", os.Getppid(), os.Args)
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
	capi.AllocCAppT().Bind(app)
	defer app.GetCAppT().UnbindAll()

	capi.AllocCBrowserProcessHandlerT().Bind(app)
	defer app.GetCBrowserProcessHandlerT().UnbindAll()

	capi.AllocCRenderProcessHandlerT().Bind(app)
	defer app.GetCBrowserProcessHandlerT().UnbindAll()

	capi.AllocCLoadHandlerT().Bind(app)
	defer app.GetCLoadHandlerT().UnbindAll()

	cef.ExecuteProcess(mainArgs, app.GetCAppT()) // Exit if this is render process

	s := capi.NewCSettingsT()
	s.SetLogSeverity(capi.LogseverityWarning)
	s.SetNoSandbox(false)
	s.SetMultiThreadedMessageLoop(false)
	s.SetRemoteDebuggingPort(8088)
	cef.Initialize(mainArgs, s, app.GetCAppT())

	capi.RunMessageLoop()

	capi.Shutdown()
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

type myBrowserProcessHandler struct {
	capi.RefToCBrowserProcessHandlerT
}

func init() {
	var _ capi.OnContextInitializedHandler = (*myBrowserProcessHandler)(nil)
}

const internalHostname = "cefingo.internal"

func (bph *myBrowserProcessHandler) GetBrowserProcessHandler(*capi.CAppT) *capi.CBrowserProcessHandlerT {
	return bph.GetCBrowserProcessHandlerT()
}

func (bph *myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("T41:")
	// Register the custom scheme handler factory.
	// RegisterSchemeHandlerFactory()

	factory := capi.AllocCSchemeHandlerFactoryT().Bind(&mySchemeHandlerFactory{})
	capi.RegisterSchemeHandlerFactory("http", internalHostname, factory)

	client := &myClient{}
	capi.AllocCClientT().Bind(client)
	capi.AllocCLifeSpanHandlerT().Bind(client)

	windowInfo := capi.NewCWindowInfoT()
	windowInfo.SetStyle(win32api.WsOverlappedwindow | win32api.WsClipchildren |
		win32api.WsClipsiblings | win32api.WsVisible)
	windowInfo.SetParentWindow(nil)
	windowInfo.SetX(win32api.CwUsedefault)
	windowInfo.SetY(win32api.CwUsedefault)
	windowInfo.SetWidth(win32api.CwUsedefault)
	windowInfo.SetHeight(win32api.CwUsedefault)
	windowInfo.SetWindowName("Cefingo Monaco Editor Example")

	browserSettings := capi.NewCBrowserSettingsT()

	capi.BrowserHostCreateBrowser(
		windowInfo,
		client.GetCClientT(),
		"http://"+internalHostname+"/main",
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

type myLifeSpanHandler struct {
	capi.RefToCLifeSpanHandlerT
}

func (lsh *myLifeSpanHandler) GetLifeSpanHandler(*capi.CClientT) *capi.CLifeSpanHandlerT {
	return lsh.GetCLifeSpanHandlerT()
}

func init() {
	var _ capi.OnAfterCreatedHandler = (*myLifeSpanHandler)(nil)
	var _ capi.OnBeforeCloseHandler = (*myLifeSpanHandler)(nil)
}

func (*myLifeSpanHandler) OnAfterCreated(self *capi.CLifeSpanHandlerT, browser *capi.CBrowserT) {
	defer browser.ForceUnref()
	capi.Logf("T68:")
}

func (lsh *myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, browser *capi.CBrowserT) {
	defer browser.ForceUnref()
	capi.Logf("T72:")
	capi.QuitMessageLoop()
	if client, ok := self.Handler().(*myClient); ok {
		capi.Logf("L124:")
		client.GetCClientT().UnbindAll()
	}
	self.UnbindAll()
}

type mySchemeHandlerFactory struct {
}

var main_text = e.Html5(e.Attr(a.Lang("ja")),
	e.Head_(
		e.Title_("monaco-editor"),
		e.Meta(e.Attr(a.Charset_("utf-8"))),
		e.Meta(e.Attr(
			a.HttpEquiv_("X-UA-Compatible"),
			a.Content_("IE=edge"))),
		e.Meta(e.Attr(
			a.HttpEquiv_("Content-Type"),
			a.Content_("text/html;charset=utf-8"))),
	),
	e.Body_(
		e.Div(e.Attr(
			a.Id_("container"),
			a.Style_("width:800px;height:400px;border:1px solid grey"),
		)),
		e.Script(e.Attr(a.Src_("/vs/loader.js")), e.JavaScript_("")),
	),
)

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
	defer browser.ForceUnref()
	url, err := url.Parse(request.GetUrl())
	if err != nil {
		capi.Logf("T356: err:%v", err)
		return nil
	}
	if url.Hostname() == internalHostname {
		if url.Path == "/main" {
			handler = capi.AllocCResourceHandlerT().Bind(&myResourceHandler{
				url:   url,
				mime:  "text/html",
				bytes: []byte(main_text),
			})
		} else if strings.HasPrefix(url.Path, "/vs/") {
			content, err := monacoPkg.ReadFile("package/min" + url.Path)
			if err != nil {
				capi.Panicf("T155: %s, %v", url.Path, err)
			}
			capi.Logf("T181: %s", url.Path)
			handler = capi.AllocCResourceHandlerT().Bind(&myResourceHandler{
				url:   url,
				mime:  ftMime(url.Path),
				bytes: content,
			})
		} else {
			capi.Logf("T132: Not Found: %s, %s", url.Hostname(), url.Path)
			handler = capi.AllocCResourceHandlerT().Bind(&notFoundHandler{
				url: url,
			})
		}
	}
	return handler
}

func ftMime(fn string) (mime string) {
	pos := strings.LastIndex(fn, ".")
	switch fn[pos:] {
	case ".js":
		mime = "text/javascript"
	case ".css":
		mime = "text/css"
	default:
		mime = "text/plain"
	}
	return mime
}

type myResourceHandler struct {
	url    *url.URL
	status int
	bytes  []byte
	next   int
	mime   string
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
	// capi.Logf("T339: %s", request.GetUrl())
	callback.Cont()
	return true
}

func (rh *myResourceHandler) GetResponseHeaders(
	self *capi.CResourceHandlerT,
	response *capi.CResponseT,
) (response_length int64, redirectUrl string) {
	capi.Logf("T391: %s: %d", rh.url.Path, rh.status)
	response.SetMimeType(rh.mime)
	// h := []capi.StringMap{
	// 	{Key: "Content-Type", Value: rh.mime + "; charset=utf-8"},
	// }
	status := rh.status
	text := ""
	if status == 0 {
		status = 200
		text = "OK"
	}
	response.SetStatus(status)
	response.SetStatusText(text)
	// response.SetHeaderMap(h)
	h := cef.NewStringMultimap()
	capi.StringMultimapAppend(h.CefObject(), "Content-Type", rh.mime+"; charset=utf-8")
	response.SetHeaderMap(h.CefObject())

	return int64(len(rh.bytes)), ""
	// response.DumpHeaders()
}

func (rh *myResourceHandler) Read(
	self *capi.CResourceHandlerT,
	data_out []byte,
	callback *capi.CResourceReadCallbackT,
) (ret bool, bytes_read int) {
	l := min(len(data_out), len(rh.bytes)-rh.next)
	capi.Logf("T214: %s %d: %d, %d", rh.url, len(data_out), len(rh.bytes), l)
	for i := 0; i < l; i++ {
		data_out[i] = rh.bytes[rh.next+i]
	}
	rh.next = rh.next + l
	bytes_read = l
	ret = true
	if l <= 0 {
		ret = false
	}
	return ret, bytes_read
}

type notFoundHandler struct {
	url  *url.URL
	text string
}

func init() {
	var _ capi.GetResponseHeadersHandler = (*notFoundHandler)(nil)
	var _ capi.ReadResponseHandler = (*notFoundHandler)(nil)
}
func (nfh *notFoundHandler) GetResponseHeaders(
	self *capi.CResourceHandlerT,
	response *capi.CResponseT,
) (response_length int64, redirectUrl string) {
	mime := "text/plain"
	response.SetMimeType(mime)
	// h := []capi.StringMap{
	// 	{Key: "Content-Type", Value: mime + "; charset=utf-8"},
	// }
	response.SetStatus(404)
	response.SetStatusText("Not Found")
	// response.SetHeaderMap(h)
	h := cef.NewStringMultimap()
	capi.StringMultimapAppend(h.CefObject(), "Content-Type", mime+"; charset=utf-8")
	response.SetHeaderMap(h.CefObject())

	nfh.text = nfh.url.Path + " Not Found."
	response_length = int64(len(nfh.text))
	// response.DumpHeaders()
	return response_length, ""
}

func (nfh *notFoundHandler) ReadResponse(
	self *capi.CResourceHandlerT,
	data_out []byte,
	callback *capi.CCallbackT,
) (ret bool, bytes_read int) {
	l := len(nfh.text)
	buf := []byte(nfh.text)
	l = min(l, len(data_out))
	for i, b := range buf[:l] {
		data_out[i] = b
	}
	bytes_read = l
	capi.Logf("T409: %d, %d", len(nfh.text), l)
	return true, bytes_read
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type myRenderProcessHandler struct {
	capi.RefToCRenderProcessHandlerT
	myLoadHandler
}

func init() {
	var _ capi.CRenderProcessHandlerTGetLoadHandlerHandler = (*myRenderProcessHandler)(nil)
}

func (rph *myRenderProcessHandler) GetRenderProcessHandler(*capi.CAppT) *capi.CRenderProcessHandlerT {
	return rph.GetCRenderProcessHandlerT()
}

type myLoadHandler struct {
	capi.RefToCLoadHandlerT
}

func (lh *myLoadHandler) GetLoadHandler(*capi.CRenderProcessHandlerT) *capi.CLoadHandlerT {
	return lh.GetCLoadHandlerT()
}

func init() {
	var _ capi.OnLoadEndHandler = (*myLoadHandler)(nil)
}

func (*myLoadHandler) OnLoadEnd(
	loadHandler *capi.CLoadHandlerT,
	browser *capi.CBrowserT,
	frame *capi.CFrameT,
	httpStatusCode int,
) {
	defer browser.ForceUnref()
	context := frame.GetV8context()
	url, _ := url.Parse(frame.GetUrl()) //
	if url.Path != "/main" {
		capi.Logf("T283: Not Handled LoadEnd: httpCode:%d, %s", httpStatusCode, frame.GetUrl())
	}

	if context.Enter() {
		defer context.Exit()

		c, err := v8.GetCurrentContext()
		if err != nil {
			capi.Logf("E292: %+v", err)
			return
		}
		require, err := c.Global.GetValueBykey("require")
		if err != nil {
			capi.Logf("E297: %+v", err)
			return
		}

		// require.config({ paths: { 'vs': '/vs'} });
		vs := v8.NewObject()
		vs.SetValueBykey("vs", v8.NewString("/vs"))
		o := v8.NewObject()
		o.SetValueBykey("paths", vs)

		if _, err := require.GetValueBykey("config"); err != nil {
			capi.Panicf("T352: can not get config")
		}
		_, err = require.Call("config", []v8.Value{o})
		if err != nil {
			capi.Panicf("E306: %+v", err)
			return
		}

		// require(['vs/editor/editor.main'], function() {
		//   var editor = monaco.editor.create(
		//     document.getElementById('container'),
		//     {
		//       value: [
		//         'function x() {',
		//         '\tconsole.log("Hello world!");',
		//         '}'
		//       ].join('\n'),
		//       language: 'javascript'
		//     });
		// });
		p1 := c.NewArray(v8.NewString("vs/editor/editor.main"))
		if err != nil {
			capi.Panicf("E315: %+v", err)
		}
		_, err = require.ExecuteFunction(v8.Value{}, []v8.Value{p1, v8.NewFunction("main", v8.HandlerFunction(
			func(this v8.Value, args []v8.Value) (v v8.Value, err error) {
				monaco, _ := c.Global.GetValueBykey("monaco")
				editor, _ := monaco.GetValueBykey("editor")
				container, _ := c.GetElementById("container")

				o := v8.NewObject()
				o.SetValueBykey("value", v8.NewString("function x(){\n\tconsole.Log('');\n}"))
				o.SetValueBykey("language", v8.NewString("javascript"))
				v, err = editor.Call("create", []v8.Value{container, o})
				capi.Logf("T330: %v", err)
				return v, err
			})),
		})
		if err != nil {
			capi.Panicf("E324: %+v", err)
		}
	}
}
