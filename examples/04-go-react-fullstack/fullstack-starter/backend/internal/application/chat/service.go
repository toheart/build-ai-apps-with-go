package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/domain/conversation"
)

var (
	ErrConversationNotFound   = errors.New("conversation not found")
	ErrConversationExists     = errors.New("conversation already exists")
	ErrConversationIDGenerate = errors.New("generate conversation id")
)

type Repository interface {
	Create(ctx context.Context, conv conversation.Conversation) error
	Load(ctx context.Context, id string) (conversation.Conversation, error)
	Save(ctx context.Context, conv conversation.Conversation) error
}

type Responder interface {
	Reply(ctx context.Context, messages []conversation.Message) (string, error)
}

type Service struct {
	repo      Repository
	responder Responder
	idgen     func() (string, error)
}

func NewService(repo Repository, responder Responder) *Service {
	return NewServiceWithGenerator(repo, responder, generateConversationID)
}

func NewServiceWithGenerator(
	repo Repository,
	responder Responder,
	idgen func() (string, error),
) *Service {
	return &Service{
		repo:      repo,
		responder: responder,
		idgen:     idgen,
	}
}

func (s *Service) CreateConversation(ctx context.Context) (conversation.Conversation, error) {
	for range 3 {
		id, err := s.idgen()
		if err != nil {
			return conversation.Conversation{}, fmt.Errorf("%w: %v", ErrConversationIDGenerate, err)
		}

		conv, err := conversation.New(id)
		if err != nil {
			return conversation.Conversation{}, fmt.Errorf("create conversation aggregate: %w", err)
		}

		if err := s.repo.Create(ctx, conv); err != nil {
			if errors.Is(err, ErrConversationExists) {
				continue
			}

			return conversation.Conversation{}, fmt.Errorf("create conversation: %w", err)
		}

		return conv, nil
	}

	return conversation.Conversation{}, fmt.Errorf("create conversation: %w", ErrConversationExists)
}

func (s *Service) GetConversation(
	ctx context.Context,
	conversationID string,
) (conversation.Conversation, error) {
	if _, err := conversation.New(conversationID); err != nil {
		return conversation.Conversation{}, fmt.Errorf("validate conversation id: %w", err)
	}

	conv, err := s.repo.Load(ctx, conversationID)
	if err != nil {
		return conversation.Conversation{}, fmt.Errorf("load conversation: %w", err)
	}

	return conv, nil
}

func (s *Service) SendMessage(
	ctx context.Context,
	conversationID string,
	userInput string,
) (conversation.Conversation, error) {
	conv, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return conversation.Conversation{}, err
	}

	if err := conv.AddUserMessage(userInput); err != nil {
		return conversation.Conversation{}, fmt.Errorf("append user message: %w", err)
	}

	reply, err := s.responder.Reply(ctx, conv.Messages())
	if err != nil {
		return conversation.Conversation{}, fmt.Errorf("generate assistant reply: %w", err)
	}

	if err := conv.AddAssistantMessage(reply); err != nil {
		return conversation.Conversation{}, fmt.Errorf("append assistant message: %w", err)
	}

	if err := s.repo.Save(ctx, conv); err != nil {
		return conversation.Conversation{}, fmt.Errorf("save conversation: %w", err)
	}

	return conv, nil
}

func generateConversationID() (string, error) {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return "conv-" + hex.EncodeToString(randomBytes), nil
}
