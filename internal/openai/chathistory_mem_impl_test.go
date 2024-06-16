package openai_test

import (
	"context"
	"testing"

	"github.com/onemouth/golinegpt/internal/openai"
	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatHistoryMemImpl(t *testing.T) {
	t.Parallel()

	var ch openai.ChatHistory = openai.NewChatHistoryMemImpl()

	mockUserID := "test-user-id"
	mockCtx := context.Background()

	t.Run("get empty", func(t *testing.T) {
		ret, err := ch.Get(mockCtx, mockUserID)
		assert.NoError(t, err)
		assert.Len(t, ret, 0)
	})

	t.Run("append 1 item and then get", func(t *testing.T) {
		err := ch.Append(mockCtx, mockUserID, []goopenai.ChatCompletionMessage{
			{Role: "system"},
		})
		assert.NoError(t, err)
		ret, err := ch.Get(mockCtx, mockUserID)
		assert.NoError(t, err)
		require.Len(t, ret, 1)
		assert.Equal(t, ret[0].Role, "system")
	})

	t.Run("append 1 item and then get again", func(t *testing.T) {
		err := ch.Append(mockCtx, mockUserID, []goopenai.ChatCompletionMessage{
			{Role: "user"},
		})
		assert.NoError(t, err)
		ret, err := ch.Get(mockCtx, mockUserID)
		assert.NoError(t, err)
		require.Len(t, ret, 2)
		assert.Equal(t, ret[1].Role, "user")
	})

	t.Run("reset and get", func(t *testing.T) {
		err := ch.Reset(mockCtx, mockUserID)
		assert.NoError(t, err)

		ret, err := ch.Get(mockCtx, mockUserID)
		assert.NoError(t, err)
		assert.Len(t, ret, 0)
	})
}
