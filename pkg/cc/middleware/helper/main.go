package helper

import (
	"net/http"

	mw "github.com/cyf-gh/ccgo/pkg/cc/middleware"
	mwu "github.com/cyf-gh/ccgo/pkg/cc/middleware/util"
)

func WrapPost(handler http.HandlerFunc) http.HandlerFunc {
	return mw.HandlerWrapFully(handler, mwu.Method(mwu.POST))
}

func WrapGet(handler http.HandlerFunc) http.HandlerFunc {
	return mw.HandlerWrapFully(handler, mwu.Method(mwu.GET))
}

func WrapWS(handler http.HandlerFunc) http.HandlerFunc {
	return mw.HandlerWrapFully(handler, mwu.Method(mwu.WS))
}
