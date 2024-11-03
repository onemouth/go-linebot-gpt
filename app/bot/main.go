package main

import (
	"cmp"
	"log/slog"
	"net/http"
	"os"

	"github.com/onemouth/golinegpt/app/bot/http/handler"
	myhttp "github.com/onemouth/golinegpt/internal/http"
	"github.com/onemouth/golinegpt/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func setLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
}

func main() {
	setLogger()

	openaiClient := openai.NewClient(option.WithAPIKey(os.Getenv("OPENAI_API_KEY")))

	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	bot, err := messaging_api.NewMessagingApiAPI(
		os.Getenv("LINE_CHANNEL_TOKEN"),
	)
	if err != nil {
		slog.Error("failed to setup line API", slog.Any("err", err))
		return
	}

	blobAPI, err := messaging_api.NewMessagingApiBlobAPI(
		os.Getenv("LINE_CHANNEL_TOKEN"),
	)
	if err != nil {
		slog.Error("failed to setup line BlobAPI", slog.Any("err", err))
		return
	}

	lineWebhookHandler := handler.NewLineWebhookHandler(
		channelSecret, bot, blobAPI, openaiClient,
	)

	mux := http.NewServeMux()

	lineSigVerifier := line.NewRequestSignatureVerifier(channelSecret)

	mux.Handle("POST /webhook", myhttp.Chain([]myhttp.Middleware{lineSigVerifier}, lineWebhookHandler))

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	port := cmp.Or(os.Getenv("PORT"), "3000")
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("http.ListenAndServe failed", slog.Any("err", err))

		return
	}
}
