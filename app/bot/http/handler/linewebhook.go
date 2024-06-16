package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	goopenai "github.com/sashabaranov/go-openai"
)

const (
	TRANSLATOR_PROMPT = `
	You are a translator. Let's work out the translation step by step.
	For the input, you will translate it into English(US), Japanese, 繁體中文(Taiwan).
	And you will output each language's result.

	If you are not sure about something in the translation result, you must add comments at the end.

	If it is hard to translate, you can just describe it.
	`
)

func chatMessageComplete(client *goopenai.Client, message string) (goopenai.ChatCompletionResponse, error) {
	return client.CreateChatCompletion(
		context.Background(),
		goopenai.ChatCompletionRequest{
			Model: goopenai.GPT4Turbo,
			Messages: []goopenai.ChatCompletionMessage{
				{
					Role:    goopenai.ChatMessageRoleSystem,
					Content: TRANSLATOR_PROMPT,
				},
				{
					Role:    goopenai.ChatMessageRoleUser,
					Content: message,
				},
			},
		},
	)
}

func chatImageComplete(client *goopenai.Client, imageURL string) (goopenai.ChatCompletionResponse, error) {
	return client.CreateChatCompletion(
		context.Background(),
		goopenai.ChatCompletionRequest{
			Model: goopenai.GPT4Turbo,
			Messages: []goopenai.ChatCompletionMessage{
				{
					Role:    goopenai.ChatMessageRoleSystem,
					Content: TRANSLATOR_PROMPT,
				},
				{
					Role: goopenai.ChatMessageRoleUser,
					MultiContent: []goopenai.ChatMessagePart{
						{
							Type:     goopenai.ChatMessagePartTypeImageURL,
							ImageURL: &goopenai.ChatMessageImageURL{URL: imageURL},
						},
					},
				},
			},
		},
	)
}

func getImageMessageURL(blobAPI *messaging_api.MessagingApiBlobAPI, imageMsg webhook.ImageMessageContent) (string, error) {
	if imageMsg.ContentProvider.Type == webhook.ContentProviderTYPE_LINE {
		resp, err := blobAPI.GetMessageContent(imageMsg.Id)
		if err != nil {
			return "", err
		}

		defer resp.Body.Close()

		newFile, err := os.Create("static/" + imageMsg.Id + ".jpg")
		if err != nil {
			return "", err
		}
		defer newFile.Close()
		io.Copy(newFile, resp.Body)

		return "https://linebot-dev.ichiban.day" + "/static/" + imageMsg.Id + ".jpg", nil

	} else if imageMsg.ContentProvider.Type == webhook.ContentProviderTYPE_EXTERNAL {
		return imageMsg.ContentProvider.OriginalContentUrl, nil
	}
	return "", fmt.Errorf("unknown content provider type %s", imageMsg.ContentProvider.Type)
}

type LineWebhookHandler struct {
	channelSecret string
	bot           *messaging_api.MessagingApiAPI
	blobAPI       *messaging_api.MessagingApiBlobAPI
	openClient    *goopenai.Client
}

func NewLineWebhookHandler(
	channelSecret string, bot *messaging_api.MessagingApiAPI, blobAPI *messaging_api.MessagingApiBlobAPI, openClient *goopenai.Client,
) LineWebhookHandler {
	return LineWebhookHandler{
		channelSecret: channelSecret,
		bot:           bot,
		blobAPI:       blobAPI,
		openClient:    openClient,
	}
}

func (im LineWebhookHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	slog.Debug("/callback called...")

	cb, err := webhook.ParseRequest(im.channelSecret, req)
	if err != nil {
		slog.Error("Cannot parse request: %+v\n", err)
		if errors.Is(err, webhook.ErrInvalidSignature) {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	slog.Debug("Handling events...")
	for _, event := range cb.Events {
		slog.Debug("/callback called...", slog.Any("event", event))

		switch e := event.(type) {
		case webhook.MessageEvent:
			switch message := e.Message.(type) {
			case webhook.TextMessageContent:
				if _, err = im.bot.ReplyMessage(
					&messaging_api.ReplyMessageRequest{
						ReplyToken: e.ReplyToken,
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

				chatResp, err := chatMessageComplete(im.openClient, message.Text)
				if err != nil {
					slog.Error("chatMessageComplete failed", slog.Any("err", err))

					continue
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
			case webhook.ImageMessageContent:
				if _, err = im.bot.ReplyMessage(
					&messaging_api.ReplyMessageRequest{
						ReplyToken: e.ReplyToken,
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

				imageURL, err := getImageMessageURL(im.blobAPI, message)
				if err != nil {
					slog.Error("getImageMessageURL failed", slog.Any("err", err))

					continue
				}

				chatResp, err := chatImageComplete(im.openClient, imageURL)
				if err != nil {
					slog.Error("chatImageComplete failed", slog.Any("err", err), slog.String("imageURL", imageURL))

					continue
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

			default:
				slog.Warn("Unsupported message content", slog.String("event_type", event.GetType()))
			}
		default:
			slog.Warn("Unsupported message", slog.Any("event", event))
		}
	}
}
