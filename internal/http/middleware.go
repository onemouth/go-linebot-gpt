package http

import "net/http"

type Middleware interface {
	Decorate(http.Handler) http.Handler
}

func Chain(middlewares []Middleware, h http.Handler) http.Handler {
	ret := h

	if len(middlewares) == 0 {
		return ret
	}

	for i := len(middlewares) - 1; i >= 0; i-- {
		ret = middlewares[i].Decorate(ret)
	}

	return ret
}
