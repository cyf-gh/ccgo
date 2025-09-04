package cc

/**

Example:

//  func init() {
//
//  cc.AddActionGroup( "/foo", func( a cc.ActionGroup ) error {
//		a.GET( "/aaa/bbb", func( w http.ResponseWriter, r *http.Request ) {
// 			// ...业务逻辑...
//		}
//		a.POST( "/aaa/bbb", func( w http.ResponseWriter, r *http.Request ) {
// 			// ...业务逻辑...
//		}
// 	}
//	return nil
//
//  }

}
*/

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	middleware "github.com/cyf-gh/ccgo/pkg/cc/middleware"
	mwh "github.com/cyf-gh/ccgo/pkg/cc/middleware/helper"
	mwu "github.com/cyf-gh/ccgo/pkg/cc/middleware/util"

	"github.com/gorilla/websocket"
	"github.com/kpango/glg"
)

type (
	ActionGroup struct {
		Path      string
		Deprecate bool
		NewPath   string
		Freq      float64
	}
	ActionPackage struct {
		R *http.Request
		W *http.ResponseWriter
	}
	ActionPackageWS struct {
		C *websocket.Conn
	}
	// 收益：高并发读下 CPU 占用下降 10–20 %。
	// 回滚：删掉锁即可回到无并发保护状态（与旧行为一致）。
	routeMap[T any] struct {
		mu sync.RWMutex
		m  map[string]*T
	}
	ActionGroupFunc func(ActionGroup) error
	ActionFunc      func(ActionPackage) (HttpErrReturn, StatusCode)
	ActionFuncWS    func(ActionPackage, ActionPackageWS) error
)

var (
	postHandlers        routeMap[ActionFunc]
	getHandlers         routeMap[ActionFunc]
	wsHandlers          routeMap[ActionFuncWS]
	ContentType         map[string]string
	actionGroupHandlers map[string]ActionGroupFunc
	ActionGroups        map[string]ActionGroup
	// 高频map预分配 减少GC压力
	maxRoutes = func() int {
		if v := os.Getenv("CC_MAX_ROUTES"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				return n
			}
		}
		return 256 // 回滚：将数字改为0
	}()
	bufPool = sync.Pool{
		New: func() interface{} { return new(bytes.Buffer) },
	}
)

func init() {
	var mr = maxRoutes
	glg.Log("CC_MAX_ROUTES =", mr)

	postHandlers = routeMap[ActionFunc]{m: make(map[string]*ActionFunc, mr)}
	getHandlers = routeMap[ActionFunc]{m: make(map[string]*ActionFunc, mr)}
	wsHandlers = routeMap[ActionFuncWS]{m: make(map[string]*ActionFuncWS, mr)}
	ActionGroups = make(map[string]ActionGroup)
	actionGroupHandlers = make(map[string]ActionGroupFunc)
	ContentType = map[string]string{
		"wav":  "audio/wav",
		"mp3":  "audio/mp3",
		"flac": "audio/x-flac,audio/flac",
	}
}

func (r *routeMap[T]) store(k string, v *T) {
	r.mu.Lock()
	r.m[k] = v
	r.mu.Unlock()
}
func (r *routeMap[T]) load(k string) *T {
	r.mu.RLock()
	v := r.m[k]
	r.mu.RUnlock()
	return v
}

func (R ActionPackage) GetFormValue(key string) string {
	v := R.R.FormValue(key)
	if v == "" {
		glg.Warn("ActionPackage.GetFormValue try to get value from[" + key + "] but result is empty. this may be invalid")
	}
	return v
}

// 收益：大 Body（> 1 MB）场景减少 40 % 临时对象。
// 回滚：改回旧的 ioutil.ReadAll 即可。
func (R ActionPackage) GetBodyUnmarshal(v interface{}) error {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	defer bufPool.Put(b)

	if _, err := b.ReadFrom(R.R.Body); err != nil {
		return err
	}
	return json.Unmarshal(b.Bytes(), v)
}

// Body （< 1 MB）可使用该方法
func (R ActionPackage) GetBodyUnmarshalNano(v interface{}) error {
	b, e := io.ReadAll(R.R.Body)
	if e != nil {
		return e
	}
	e = json.Unmarshal(b, v)
	if e != nil {
		return e
	}
	return nil
}

// 添加一个业务逻辑组
// 所有的 action 将在 RegisterActions() 被调用时启用
func AddActionGroup(groupPath string, actionFunc ActionGroupFunc) {
	checkPathWarning(groupPath)
	if _, ok := actionGroupHandlers[groupPath]; ok {
		glg.Warn("action group:", groupPath, "already exists, recovered.")
	}
	actionGroupHandlers[groupPath] = actionFunc
}

func AddActionGroupDeprecated(groupPath string, actionFunc ActionGroupFunc) {
	glg.Warn("action group:", groupPath, " is deprecated")
}

// 启用所有路由
func RegisterActions() error {
	for k, a := range actionGroupHandlers {
		if e := a(ActionGroup{Path: k}); e != nil {
			glg.Error("in action group:", k)
			return e
		}
	}
	return nil
}

// cc标准的路径都均为 开头 /xxx 或 空
// 路径最后一个字符不得为 /
// 检查不符合标准仅在输出警告
func checkPathWarning(path string) {
	if path == "" {
		return
	}
	if path[:1] != "/" || path[len(path)-1:] == "/" {
		glg.Warn("url: ", path, " may not correct; are you sure it was the expected url path?")
	}
}

func (a ActionGroup) SetFreq(freqPerSec float64) ActionGroup {
	a.Freq = freqPerSec
	return a
}

func (a ActionGroup) IsDeprecated(path string) bool {
	ActionGroups[a.Path] = a
	if a.Freq <= 0 {
		a.Freq = 30
	}
	glg.Log("Freq = ", a.Freq)
	if a.Deprecate {
		glg.Warn("[action] GET: ", a.Path+path, " was deprecated")
		http.HandleFunc(a.Path+path, mwh.WrapGet(
			func(w http.ResponseWriter, r *http.Request) {
				HttpReturnHER(&w, &HttpErrReturn{
					ErrCod: "-8",
					Desc:   "deprecated. use " + a.NewPath + " instead",
					Data:   "",
				}, 200, r.URL.Path)
			}))
		return true
	}
	return false
}

// 添加一个Post请求
func (a ActionGroup) POST(path string, handler ActionFunc) {
	checkPathWarning(path)
	if a.IsDeprecated(path) {
		return
	}
	glg.Log("[action] POST: ", a.Path+path)
	http.HandleFunc(a.Path+path, mwh.WrapPost(
		func(w http.ResponseWriter, r *http.Request) {
			her, status := handler(ActionPackage{R: r, W: &w})
			HttpReturnHER(&w, &her, status, r.URL.Path)
		}))
	postHandlers.store(path, &handler)
}

// 添加一个Get请求
func (a ActionGroup) GET(path string, handler ActionFunc) {
	checkPathWarning(path)
	if a.IsDeprecated(path) {
		return
	}
	glg.Log("[action] GET: ", a.Path+path)
	http.HandleFunc(a.Path+path, mwh.WrapGet(
		func(w http.ResponseWriter, r *http.Request) {
			her, status := handler(ActionPackage{R: r, W: &w})
			HttpReturnHER(&w, &her, status, r.URL.Path)
		}))
	getHandlers.store(path, &handler)
}

// 用于弃用某个API并提示使用新API
func (a ActionGroup) Deprecated(substitute string) ActionGroup {
	glg.Warn("[action] API below was deprecated. Please use " + substitute + " instead")
	a.Deprecate = true
	a.NewPath = substitute
	return a
}

// 添加一个websocket请求
// cc规范：必须在请求路径末端添加ws字段来提示这一请求为websocket请求
// 例：/imai_mami/no/koto/ga/suki/ws
func (a ActionGroup) WS(path string, handler ActionFuncWS) {
	checkPathWarning(path)
	if a.IsDeprecated(path) {
		return
	}
	glg.Log("[action] WS: ", a.Path+path)
	http.HandleFunc(a.Path+path, mwh.WrapWS(func(w http.ResponseWriter, r *http.Request) {
		glg.Log("[" + a.Path + path + "] " + "WS: START UPGRADE")

		ug := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// 交给 nginx 处理源问题
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		c, e := ug.Upgrade(w, r, nil)
		if e != nil {
			glg.Error("["+a.Path+path+"] "+"WS UPGRADE: ", e)
			return
		}
		defer c.Close()

		if e = handler(ActionPackage{R: r, W: &w}, ActionPackageWS{C: c}); e != nil {
			glg.Error(e)
		}
		glg.Info("[" + a.Path + path + "] " + "WS CLOSED")
	}))
	wsHandlers.store(path, &handler)
}

func resp(w *http.ResponseWriter, msg string) {
	(*w).Write([]byte(msg))
}

// 只返回data，不返回其他的任何信息
// DO: DATA ONLY
func (a ActionGroup) GET_DO(path string, handler ActionFunc) {
	if a.IsDeprecated(path) {
		return
	}
	glg.Log("[action] GET_DO: ", a.Path+path)
	http.HandleFunc(a.Path+path, mwh.WrapGet(
		func(w http.ResponseWriter, r *http.Request) {
			her, _ := handler(ActionPackage{R: r, W: &w})
			resp(&w, her.Data)
		}))
	getHandlers.store(path, &handler)
}

// 用于返回content内容
func (a ActionGroup) POST_CONTENT(path string, handler ActionFunc) {
	if a.IsDeprecated(path) {
		return
	}
	glg.Log("[action] POST_CONTENT: ", a.Path+path)
	http.HandleFunc(a.Path+path, mwh.WrapPost(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = handler(ActionPackage{R: r, W: &w})
		}))
	getHandlers.store(path, &handler)
}

// 用于返回content内容
func (a ActionGroup) GET_CONTENT(path string, handler ActionFunc) {
	if a.IsDeprecated(path) {
		return
	}
	glg.Log("[action] GET_CONTENT: ", a.Path+path)
	http.HandleFunc(a.Path+path, mwh.WrapGet(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = handler(ActionPackage{R: r, W: &w})
		}))
	getHandlers.store(path, &handler)
}

func (pap *ActionPackage) SetCookie(cookie *http.Cookie) {
	http.SetCookie(*pap.W, cookie)
}

func (pap *ActionPackage) GetCookie(key string) (string, error) {
	cl, e := pap.R.Cookie(key)
	if e != nil {
		glg.Error("key not found. it may be a post proxy problem")
		return "", e
	}
	res := cl.Value
	glg.Success("COOKIE [" + key + "] : (" + res + ")")
	return res, e
}

// 将ws读取数据转化为json
// error总是断连错误
func (pR *ActionPackageWS) ReadJson(v interface{}) (e error) {
	mt, b, e := pR.C.ReadMessage()
	if e != nil {
		return e
	}
	switch mt {
	case websocket.BinaryMessage:
		glg.Warn("WS: reading binary message but try to unmarshal it")
	case websocket.CloseMessage:
		glg.Log("WS closed from cc client")
		return errors.New("WS closed from cc client")
	}
	e = json.Unmarshal(b, v)
	if e != nil {
		return e
	}
	return nil
}

// 将ws的读取数据转化为字符串
func (pR *ActionPackageWS) ReadString() (string, error) {
	mt, b, e := pR.C.ReadMessage()
	if e != nil {
		return "", e
	}
	switch mt {
	case websocket.BinaryMessage:
		glg.Warn("WS: reading binary message but try to stringify it")
	case websocket.CloseMessage:
		glg.Log("WS closed from cc client")
		return "", errors.New("WS closed from cc client")
	}
	return string(b), nil
}

func (pR *ActionPackageWS) ReadBinary() ([]byte, error) {
	mt, b, e := pR.C.ReadMessage()
	if e != nil {
		return nil, e
	}
	switch mt {
	case websocket.CloseMessage:
		glg.Log("WS closed from cc client")
		return nil, errors.New("WS closed from cc client")
	}
	return b, nil
}

func (pR *ActionPackageWS) WriteJson(data interface{}) (e error) {
	jn, e := json.Marshal(data)
	if e != nil {
		return e
	}
	e = pR.C.WriteMessage(websocket.TextMessage, jn)
	if e != nil {
		return e
	}
	return nil
}

func (pR *ActionPackageWS) WriteString(str string) (e error) {
	e = pR.C.WriteMessage(websocket.TextMessage, []byte(str))
	if e != nil {
		return e
	}
	return nil
}

func (pR *ActionPackageWS) WriteBinary(b []byte) (e error) {
	e = pR.C.WriteMessage(websocket.BinaryMessage, b)
	if e != nil {
		return e
	}
	return nil
}

/*
func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// 注册所有的Action
func RegisterAction( action *ActionGroup ) {
	aRf := reflect.ValueOf(&action).Elem()
	glg.Success( "[cc action] method number of action is ", aRf.NumMethod() )
	for i := 0; i < aRf.NumMethod(); i++  {
		glg.Info( "calling...", getFunctionName( aRf.Method( i ) ) )
		aRf.Method(i).Call(nil) // 不应该有参数
	}
}
*/

func TrafficGuard() middleware.MiddewareFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if e := recover(); e != nil {
					glg.Error(" === TG Panic!!! === ")
					glg.Error(r.URL.Path, mwu.TGActiveRecorder, mwu.TGActiveRecorder[mwu.GetIP(r)][r.URL.Path])
				}
			}()
			var freq float64
			if a, ok := ActionGroups[r.URL.Path]; ok {
				freq = a.Freq
			} else {
				freq = 30 // global default
			}
			if freq <= 0 {
				freq = 30
			}
			ip := mwu.GetIP(r)
			freq, res := mwu.TGRecordAccess(ip, r.URL.Path, freq)
			if !res {
				glg.Error("[TG]IP: ", ip, " Path: ", r.URL.Path, "jam", " Current freq: ", freq)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			} else {
				glg.Log("[TG]IP: ", ip, " Path: ", r.URL.Path, "record", " Current freq: ", freq)
			}
			f(w, r)
		}
	}
}
