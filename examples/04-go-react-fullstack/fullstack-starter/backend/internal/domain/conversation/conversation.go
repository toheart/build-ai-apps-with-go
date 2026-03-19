package conversation

import (
	"errors"
	"strings"
)

var (
	ErrEmptyConversationID = errors.New("conversation id is required")
	ErrEmptyMessageContent = errors.New("message content is required")
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

type Conversation struct {
	id       string
	messages []Message
}

func New(id string) (Conversation, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return Conversation{}, ErrEmptyConversationID
	}

	return Conversation{id: trimmedID}, nil
}

func Restore(id string, messages []Message) (Conversation, error) {
	conv, err := New(id)
	if err != nil {
		return Conversation{}, err
	}

	for _, message := range messages {
		if err := conv.appendMessage(message.Role, message.Content); err != nil {
			return Conversation{}, err
		}
	}

	return conv, nil
}

func (c *Conversation) ID() string {
	return c.id
}

func (c *Conversation) AddUserMessage(content string) error {
	return c.appendMessage(RoleUser, content)
}

func (c *Conversation) AddAssistantMessage(content string) error {
	return c.appendMessage(RoleAssistant, content)
}

func (c *Conversation) Messages() []Message {
	items := make([]Message, len(c.messages))
	copy(items, c.messages)

	return items
}

func (c *Conversation) appendMessage(role Role, content string) error {
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return ErrEmptyMessageContent
	}

	c.messages = append(c.messages, Message{
		Role:    role,
		Content: trimmedContent,
	})

	return nil
}
