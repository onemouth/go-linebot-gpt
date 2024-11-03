package openai

import (
	"context"

	openai "github.com/openai/openai-go"
)

type ChatHistory interface {
	Get(ctx context.Context, userID string) ([]openai.ChatCompletionMessage, error)
	Reset(ctx context.Context, userID string) error
	Append(ctx context.Context, userID string, msgs []openai.ChatCompletionMessage) error
}
