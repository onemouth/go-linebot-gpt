package openai

import (
	"context"

	goopenai "github.com/sashabaranov/go-openai"
)

type ChatHistory interface {
	Get(ctx context.Context, userID string) ([]goopenai.ChatCompletionMessage, error)
	Reset(ctx context.Context, userID string) error
	Append(ctx context.Context, userID string, msgs []goopenai.ChatCompletionMessage) error
}
