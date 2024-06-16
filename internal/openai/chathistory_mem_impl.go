package openai

import (
	"context"
	"sync"

	goopenai "github.com/sashabaranov/go-openai"
)

// chatHistoryMemImpl has a map M
// M: userID -> []goopenai.ChatCompletionMessage
type chatHistoryMemImpl struct {
	m    map[string][]goopenai.ChatCompletionMessage
	lock sync.RWMutex
}

func NewChatHistoryMemImpl() *chatHistoryMemImpl {
	return &chatHistoryMemImpl{
		m: make(map[string][]goopenai.ChatCompletionMessage),
	}
}

func (im *chatHistoryMemImpl) Get(ctx context.Context, userID string) ([]goopenai.ChatCompletionMessage, error) {
	im.lock.RLock()
	defer im.lock.RUnlock()

	v, ok := im.m[userID]
	if !ok {
		return nil, nil
	}

	return v, nil
}

func (im *chatHistoryMemImpl) Reset(ctx context.Context, userID string) error {
	im.lock.Lock()
	defer im.lock.Unlock()

	im.m[userID] = nil

	return nil
}

func (im *chatHistoryMemImpl) Append(ctx context.Context, userID string, msgs []goopenai.ChatCompletionMessage) error {
	im.lock.Lock()
	defer im.lock.Unlock()

	im.m[userID] = append(im.m[userID], msgs...)

	return nil
}
