package line

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

type CallbackRequestKey struct{}

type RequestSignatureVerifier struct {
	channelSecret string
}

func NewRequestSignatureVerifier(channelSecret string) *RequestSignatureVerifier {
	return &RequestSignatureVerifier{
		channelSecret: channelSecret,
	}
}

func (im *RequestSignatureVerifier) Decorate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(respW http.ResponseWriter, req *http.Request) {
		buf := bytes.Buffer{}
		req.Body = io.NopCloser(io.TeeReader(req.Body, &buf))

		cb, err := webhook.ParseRequest(im.channelSecret, req)
		if err != nil {
			slog.Error("Cannot parse request: %+v\n", err)
			if errors.Is(err, webhook.ErrInvalidSignature) {
				respW.WriteHeader(400)
			} else {
				respW.WriteHeader(500)
			}
			return
		}

		// reset body to buffer
		req.Body = io.NopCloser(&buf)
		// add cb to context
		ctxWithCb := context.WithValue(req.Context(), CallbackRequestKey{}, cb)

		next.ServeHTTP(respW, req.WithContext(ctxWithCb))
	})
}
