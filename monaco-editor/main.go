package main

//go:generate statik -src package/min -f

import (
	// "fmt"
	"io/ioutil"
	"log"
	"net/http"
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

	"github.com/rakyll/statik/fs"
	_ "github.com/turutcrane/cefingo-sample/monaco-editor/statik"
)

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

	app := capi.AllocCAppT()

	bph := capi.AllocCBrowserProcessHandlerT().Bind(&myBrowserProcessHandler{})
	app.AssocBrowserProcessHandlerT(bph)

	rph := capi.AllocCRenderProcessHandlerT().Bind(&myRenderProcessHandler{})
	lh := capi.AllocCLoadHandlerT().Bind(&myLoadHandler{})
	rph.AssocLoadHandlerT(lh)
	app.AssocRenderProcessHandlerT(rph)

	capi.ExecuteProcess(app) // Exit if this is render process

	s := capi.Settings{}
	s.LogSeverity = capi.LogseverityWarning // C.LOGSEVERITY_WARNING // Show only warnings/errors
	s.NoSandbox = 0
	s.MultiThreadedMessageLoop = 0
	s.RemoteDebuggingPort = 8088 // enabled if 1024-65535
	capi.Initialize(s, app)

	capi.RunMessageLoop()

	capi.Shutdown()
}

type myBrowserProcessHandler struct {
}

const internalHostname = "cefingo.internal"

func init() {
	var _ capi.OnContextInitializedHandler = &myBrowserProcessHandler{}
}

func (bph *myBrowserProcessHandler) OnContextInitialized(sef *capi.CBrowserProcessHandlerT) {
	capi.Logf("T41:")
	// Register the custom scheme handler factory.
	// RegisterSchemeHandlerFactory()

	factory := capi.AllocCSchemeHandlerFactoryT().Bind(&mySchemeHandlerFactory{})
	capi.RegisterSchemeHandlerFactory("http", internalHostname, factory)

	client := capi.AllocCClientT().Bind(&myClient{})
	life_span_handler :=
		capi.AllocCLifeSpanHandlerT().Bind(&myLifeSpanHandler{})
	client.AssocLifeSpanHandlerT(life_span_handler)

	capi.BrowserHostCreateBrowser("Cefingo Example", "http://"+internalHostname+"/main", client)
}

type myClient struct {
}

type myLifeSpanHandler struct {
}

func init() {
	var _ capi.OnAfterCreatedHandler = &myLifeSpanHandler{}
	var _ capi.OnBeforeCloseHandler = &myLifeSpanHandler{}
}

func (*myLifeSpanHandler) OnAfterCreated(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.Logf("T68:")
}

func (*myLifeSpanHandler) OnBeforeClose(self *capi.CLifeSpanHandlerT, brwoser *capi.CBrowserT) {
	capi.Logf("T72:")
	capi.QuitMessageLoop()
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

var statikFs http.FileSystem

func init() {
	var err error
	statikFs, err = fs.New()
	if err != nil {
		log.Fatalln(err)
	}
}

func init() {
	var _ capi.CreateHandler = &mySchemeHandlerFactory{}
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
			f, err := statikFs.Open(url.Path)
			if err != nil {
				log.Panicf("T163: %s %v\n", url.Path, err)
			}
			content, err := ioutil.ReadAll(f)
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
	var _ capi.ProcessRequestHandler = &myResourceHandler{}
	var _ capi.GetResponseHeadersHandler = &myResourceHandler{}
	var _ capi.ReadResponseHandler = &myResourceHandler{}
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
	response_length *int64,
	redirectUrl *string,
) {
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

	*response_length = int64(len(rh.bytes))
	// response.DumpHeaders()
}

func (rh *myResourceHandler) ReadResponse(
	self *capi.CResourceHandlerT,
	data_out []byte,
	bytes_read *int,
	callback *capi.CCallbackT,
) bool {
	l := min(len(data_out), len(rh.bytes)-rh.next)
	capi.Logf("T214: %s %d: %d, %d", rh.url, len(data_out), len(rh.bytes), l)
	for i := 0; i < l; i++ {
		data_out[i] = rh.bytes[rh.next+i]
	}
	rh.next = rh.next + l
	*bytes_read = l
	return true
}

type notFoundHandler struct {
	url  *url.URL
	text string
}

func init() {
	var _ capi.GetResponseHeadersHandler = &notFoundHandler{}
	var _ capi.ReadResponseHandler = &notFoundHandler{}
}
func (nfh *notFoundHandler) GetResponseHeaders(
	self *capi.CResourceHandlerT,
	response *capi.CResponseT,
	response_length *int64,
	redirectUrl *string,
) {
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
	*response_length = int64(len(nfh.text))
	// response.DumpHeaders()
}

func (nfh *notFoundHandler) ReadResponse(
	self *capi.CResourceHandlerT,
	data_out []byte,
	bytes_read *int,
	callback *capi.CCallbackT,
) bool {
	l := len(nfh.text)
	buf := []byte(nfh.text)
	l = min(l, len(data_out))
	for i, b := range buf[:l] {
		data_out[i] = b
	}
	*bytes_read = l
	capi.Logf("T409: %d, %d", len(nfh.text), l)
	return true
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type myRenderProcessHandler struct {
}

type myLoadHandler struct {
}

func init() {
	var _ capi.OnLoadEndHandler = &myLoadHandler{}
}

func (*myLoadHandler) OnLoadEnd(
	loadHandler *capi.CLoadHandlerT,
	browser *capi.CBrowserT,
	frame *capi.CFrameT,
	httpStatusCode int,
) {
	context := frame.GetV8context()
	url, _ := url.Parse(frame.GetUrl()) //
	if url.Path != "/main" {
		capi.Logf("T283: Not Handled LoadEnd: httpCode:%d, %s", httpStatusCode, frame.GetUrl())
	}

	if context.Enter() {
		defer context.Exit()

		c, err := v8.GetContext()
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
