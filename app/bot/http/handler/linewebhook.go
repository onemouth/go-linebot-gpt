package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/onemouth/golinegpt/internal/line"
	openai "github.com/openai/openai-go"
)

const (
	TRANSLATOR_PROMPT = `
	You are a language teacher. Let's work out the translation step by step.
	For the input, you will translate it into English(US), Japanese, 繁體中文(Taiwan).
	And you will output each language's result.

	If you are not sure about something in the translation result, you must add comments at the end.

	If it is hard to translate, you can just describe it.
	`

	SPEAKER_PROMPT = `
	You are a language teacher. Let's work out the translation step by step.
	For the input, you will translate it into English(US), Japanese, 繁體中文(Taiwan).
	And you will output each language's result.

	If you are not sure about something in the translation result, you must add comments at the end.

	If it is hard to translate, you can just describe it.

	After the translation, you will expliain more the context in 繁體中文(Taiwan).
	`
)

func chatMessageComplete(ctx context.Context, client *openai.Client, message string) (*openai.ChatCompletion, error) {
	ctx = context.WithoutCancel(ctx)

	return client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.F(openai.ChatModelGPT4o),
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(TRANSLATOR_PROMPT),
			openai.UserMessage(message),
		}),
	})
}

func chatImageComplete(ctx context.Context, client *openai.Client, imageURL string) (*openai.ChatCompletion, error) {
	ctx = context.WithoutCancel(ctx)

	return client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.F(openai.ChatModelGPT4o),
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(TRANSLATOR_PROMPT),
			openai.UserMessageParts(openai.ImagePart(imageURL)),
		}),
	})
}

func getImageMessageURL(blobAPI *messaging_api.MessagingApiBlobAPI, imageMsg webhook.ImageMessageContent) (string, string, error) {
	if imageMsg.ContentProvider.Type == webhook.ContentProviderTYPE_LINE {
		resp, err := blobAPI.GetMessageContent(imageMsg.Id)
		if err != nil {
			return "", "", err
		}

		defer resp.Body.Close()

		localPath := "static/" + imageMsg.Id + ".jpg"

		newFile, err := os.Create(localPath)
		if err != nil {
			return "", "", err
		}
		defer newFile.Close()
		io.Copy(newFile, resp.Body)

		return "https://linebot-dev.ichiban.day/" + localPath, localPath, nil

	} else if imageMsg.ContentProvider.Type == webhook.ContentProviderTYPE_EXTERNAL {
		return imageMsg.ContentProvider.OriginalContentUrl, "", nil
	}
	return "", "", fmt.Errorf("unknown content provider type %s", imageMsg.ContentProvider.Type)
}

type LineWebhookHandler struct {
	channelSecret string
	bot           *messaging_api.MessagingApiAPI
	blobAPI       *messaging_api.MessagingApiBlobAPI
	openaiClient  *openai.Client
}

func NewLineWebhookHandler(
	channelSecret string, bot *messaging_api.MessagingApiAPI, blobAPI *messaging_api.MessagingApiBlobAPI, openaiClient *openai.Client,
) LineWebhookHandler {
	return LineWebhookHandler{
		channelSecret: channelSecret,
		bot:           bot,
		blobAPI:       blobAPI,
		openaiClient:  openaiClient,
	}
}

func (im LineWebhookHandler) replyFlyingMoneyMessage(replyToken string) {
	if _, err := im.bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{
					Text: "💸💸💸",
				},
			},
		},
	); err != nil {
		slog.Error("failed to reply message", slog.Any("err", err))
	} else {
		slog.Debug("Sent text reply.")
	}
}

func (im LineWebhookHandler) handleTextMessage(ctx context.Context, e webhook.MessageEvent, message webhook.TextMessageContent) {
	im.replyFlyingMoneyMessage(e.ReplyToken)

	chatResp, err := chatMessageComplete(ctx, im.openaiClient, message.Text)
	if err != nil {
		slog.Error("chatMessageComplete failed", slog.Any("err", err))

		return
	}

	userSource, _ := e.Source.(webhook.UserSource)

	if _, err = im.bot.PushMessage(
		&messaging_api.PushMessageRequest{
			To: userSource.UserId,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{
					Text:       chatResp.Choices[0].Message.Content,
					QuoteToken: message.QuoteToken,
				},
			},
		}, "",
	); err != nil {
		slog.Error("failed to reply message", slog.Any("err", err))
	} else {
		slog.Debug("Sent text reply.")
	}
}

func (im LineWebhookHandler) handleImageMessage(ctx context.Context, e webhook.MessageEvent, message webhook.ImageMessageContent) {
	im.replyFlyingMoneyMessage(e.ReplyToken)

	imageURL, localPath, err := getImageMessageURL(im.blobAPI, message)
	if err != nil {
		slog.Error("getImageMessageURL failed", slog.Any("err", err))

		return
	}

	if localPath != "" {
		defer os.Remove(localPath)
	}

	chatResp, err := chatImageComplete(ctx, im.openaiClient, imageURL)
	if err != nil {
		slog.Error("chatImageComplete failed", slog.Any("err", err), slog.String("imageURL", imageURL))

		return
	}

	userSource, _ := e.Source.(webhook.UserSource)

	if _, err = im.bot.PushMessage(
		&messaging_api.PushMessageRequest{
			To: userSource.UserId,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{
					Text:       chatResp.Choices[0].Message.Content,
					QuoteToken: message.QuoteToken,
				},
			},
		}, "",
	); err != nil {
		slog.Error("failed to reply message", slog.Any("err", err))
	} else {
		slog.Debug("Sent text reply.")
	}
}

func (im LineWebhookHandler) handleAudioMessage(ctx context.Context, e webhook.MessageEvent, message webhook.AudioMessageContent) {
	im.replyFlyingMoneyMessage(e.ReplyToken)
	slog.Debug("audio message", slog.Any("message", message))

	resp, err := im.blobAPI.GetMessageContent(message.Id)
	if err != nil {
		slog.Error("handleAudioMessage", slog.Any("err", err))

		return
	}
	defer resp.Body.Close()

	// sting buf as Writer
	var stringBuilder strings.Builder
	b64Encoder := base64.NewEncoder(base64.StdEncoding, &stringBuilder)

	teeReader := io.TeeReader(resp.Body, b64Encoder)
	_, err = io.ReadAll(teeReader)
	if err != nil {
		slog.Error("io.Readall teeReader", slog.Any("err", err))
	}
	b64Encoder.Close()

	ctx = context.WithoutCancel(ctx)

	reply, err := im.openaiClient.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model:      openai.F(openai.ChatModelGPT4oAudioPreview),
			Modalities: openai.F([]openai.ChatCompletionModality{openai.ChatCompletionModalityText, openai.ChatCompletionModalityAudio}),
			Audio: openai.F(openai.ChatCompletionAudioParam{
				Format: openai.F(openai.ChatCompletionAudioParamFormatMP3),
				Voice:  openai.F(openai.ChatCompletionAudioParamVoiceSage),
			}),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(SPEAKER_PROMPT),
				openai.UserMessage("how to convert aac to mp3 freely"),
			}),
		})
	if err != nil {
		slog.Error("genreate audio error", slog.Any("err", err))
	}

	slog.Info(reply.Choices[0].Message.Content)
	localPath := "static/" + message.Id + ".mp3"

	newFile, err := os.Create(localPath)
	if err != nil {
		return
	}
	defer newFile.Close()

	b64Decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(reply.Choices[0].Message.Audio.Data))
	io.Copy(newFile, b64Decoder)
}

func (im LineWebhookHandler) handleMessageEvent(ctx context.Context, e webhook.MessageEvent) {
	switch message := e.Message.(type) {
	case webhook.TextMessageContent:
		im.handleTextMessage(ctx, e, message)
	case webhook.ImageMessageContent:
		im.handleImageMessage(ctx, e, message)
	case webhook.AudioMessageContent:
		im.handleAudioMessage(ctx, e, message)
	default:
		slog.Warn("Unsupported message content", slog.String("event_type", e.GetType()))
	}
}

func (im LineWebhookHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	slog.Debug("/callback called...")

	ctx := req.Context()

	cbOrNil := ctx.Value(line.CallbackRequestKey{})
	cb, ok := cbOrNil.(*webhook.CallbackRequest)
	if !ok {
		slog.Error("Cannot find cb in context")

		w.WriteHeader(500)
		return
	}

	slog.Debug("Handling events...")
	for _, event := range cb.Events {
		slog.Debug("/callback called...", slog.Any("event", event))

		switch e := event.(type) {
		case webhook.MessageEvent:
			im.handleMessageEvent(ctx, e)
		default:
			slog.Warn("Unsupported message", slog.Any("event", event))
		}
	}
}
