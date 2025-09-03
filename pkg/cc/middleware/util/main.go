// 提供了基本的中间件
package middlewareUtil

import (
	"net/http"
	"time"

	"github.com/cyf-gh/ccgo/pkg/cc/middleware"

	"github.com/kpango/glg"
)

const (
	POST = "POST"
	GET  = "GET"
	WS   = "WS"
)

// 输出请求所用时间
//
// 应当最开始注册，避免遗漏中间件的所用时间
func LogUsedTime() middleware.MiddewareFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			glg.Log(r.URL.Path, "[time started recording]")
			start := time.Now()
			defer func() {
				glg.Log(r.URL.Path, "[time used]", time.Since(start))
			}()
			f(w, r)
		}
	}
}

// 启用Cookie携带
func EnableCookie() middleware.MiddewareFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			f(w, r)
		}
	}
}

// 启用跨域（测试中使用）
func EnableAllowOrigin() middleware.MiddewareFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			f(w, r)
		}
	}
}

// 限定请求方法
func Method(m string) middleware.MiddewareFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if m != WS {
				if r.Method != m {
					glg.Error("Target Method: ", r.Method, "| Register Method:", m)
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
			} else {
				glg.Log("Method WS")
			}
			f(w, r)
		}
	}
}

// 访问记录
func AccessRecord() middleware.MiddewareFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			go CheckIPInfo(r)
			f(w, r)
		}
	}
}
