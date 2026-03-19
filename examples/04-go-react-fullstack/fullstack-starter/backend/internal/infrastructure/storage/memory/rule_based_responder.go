package memory

import (
	"context"
	"errors"
	"fmt"

	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/domain/conversation"
)

var ErrNoUserMessage = errors.New("no user message available")

type RuleBasedResponder struct{}

func NewRuleBasedResponder() RuleBasedResponder {
	return RuleBasedResponder{}
}

func (r RuleBasedResponder) Reply(
	ctx context.Context,
	messages []conversation.Message,
) (string, error) {
	_ = ctx

	for idx := len(messages) - 1; idx >= 0; idx-- {
		if messages[idx].Role == conversation.RoleUser {
			return fmt.Sprintf("已收到：%s", messages[idx].Content), nil
		}
	}

	return "", ErrNoUserMessage
}
