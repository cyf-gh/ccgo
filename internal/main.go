package main

import (
	"net/http"

	"github.com/kpango/glg"

	"github.com/cyf-gh/ccgo/pkg/cc"
	mw "github.com/cyf-gh/ccgo/pkg/cc/middleware"
	mwu "github.com/cyf-gh/ccgo/pkg/cc/middleware/util"
)

func InitMiddlewares() {
	glg.Log("middleware loading...")
	mw.Register(mwu.LogUsedTime())
	mw.Register(cc.ErrorFetcher())
	mw.Register(cc.TrafficGuard())
	mw.Register(mwu.AccessRecord())
	// mw.Register( mwu.EnableCookie() )
	// mw.Register( mwu.EnableAllowOrigin() )
	glg.Log("middleware finished loading")
}

func main() {
	cc.AddActionGroup("/api", func(a cc.ActionGroup) error {
		// \brief echo
		// \note echo
		a.GET("/echo", func(ap cc.ActionPackage) (cc.HttpErrReturn, cc.StatusCode) {
			return cc.HerOkWithData(ap.GetFormValue("a"))
		})
		return nil
	})

	InitMiddlewares()

	if e := cc.RegisterActions(); e != nil {
		panic(e)
	}

	// 启动
	http.ListenAndServe(":8080", nil)
}
