package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/domain/conversation"
)

type inMemoryRepository struct {
	items map[string]conversation.Conversation
}

func newInMemoryRepository() *inMemoryRepository {
	return &inMemoryRepository{
		items: make(map[string]conversation.Conversation),
	}
}

func (r *inMemoryRepository) Create(
	_ context.Context,
	conv conversation.Conversation,
) error {
	if _, exists := r.items[conv.ID()]; exists {
		return ErrConversationExists
	}

	r.items[conv.ID()] = conv
	return nil
}

func (r *inMemoryRepository) Load(
	_ context.Context,
	id string,
) (conversation.Conversation, error) {
	item, ok := r.items[id]
	if !ok {
		return conversation.Conversation{}, ErrConversationNotFound
	}

	return item, nil
}

func (r *inMemoryRepository) Save(
	_ context.Context,
	conv conversation.Conversation,
) error {
	if _, exists := r.items[conv.ID()]; !exists {
		return ErrConversationNotFound
	}

	r.items[conv.ID()] = conv
	return nil
}

type fixedResponder struct {
	reply string
	err   error
}

func (r fixedResponder) Reply(_ context.Context, _ []conversation.Message) (string, error) {
	if r.err != nil {
		return "", r.err
	}

	return r.reply, nil
}

func TestCreateConversationPersistsEmptyConversation(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepository()
	service := NewServiceWithGenerator(
		repo,
		fixedResponder{reply: "unused"},
		func() (string, error) { return "conv-001", nil },
	)

	conv, err := service.CreateConversation(context.Background())
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	if conv.ID() != "conv-001" {
		t.Fatalf("expected conversation id conv-001, got %q", conv.ID())
	}

	if len(conv.Messages()) != 0 {
		t.Fatalf("expected empty conversation, got %d messages", len(conv.Messages()))
	}

	stored, err := repo.Load(context.Background(), "conv-001")
	if err != nil {
		t.Fatalf("load stored conversation: %v", err)
	}

	if stored.ID() != "conv-001" {
		t.Fatalf("expected stored id conv-001, got %q", stored.ID())
	}
}

func TestGetConversationReturnsMissingConversationError(t *testing.T) {
	t.Parallel()

	service := NewService(newInMemoryRepository(), fixedResponder{reply: "unused"})

	_, err := service.GetConversation(context.Background(), "conv-404")
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestSendMessageAppendsUserAndAssistantMessages(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepository()
	service := NewServiceWithGenerator(
		repo,
		fixedResponder{reply: "已收到：hi"},
		func() (string, error) { return "conv-001", nil },
	)

	conv, err := service.CreateConversation(context.Background())
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	updated, err := service.SendMessage(context.Background(), conv.ID(), "  hi  ")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}

	got := updated.Messages()
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}

	if got[0].Role != conversation.RoleUser || got[0].Content != "hi" {
		t.Fatalf("unexpected user message: %+v", got[0])
	}

	if got[1].Role != conversation.RoleAssistant || got[1].Content != "已收到：hi" {
		t.Fatalf("unexpected assistant message: %+v", got[1])
	}
}

func TestSendMessageRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepository()
	service := NewServiceWithGenerator(
		repo,
		fixedResponder{reply: "unused"},
		func() (string, error) { return "conv-001", nil },
	)

	_, err := service.CreateConversation(context.Background())
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	_, err = service.SendMessage(context.Background(), "conv-001", "   ")
	if !errors.Is(err, conversation.ErrEmptyMessageContent) {
		t.Fatalf("expected ErrEmptyMessageContent, got %v", err)
	}
}

func TestSendMessageReturnsResponderError(t *testing.T) {
	t.Parallel()

	repo := newInMemoryRepository()
	service := NewServiceWithGenerator(
		repo,
		fixedResponder{err: errors.New("model unavailable")},
		func() (string, error) { return "conv-001", nil },
	)

	_, err := service.CreateConversation(context.Background())
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	_, err = service.SendMessage(context.Background(), "conv-001", "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := err.Error(); got != "generate assistant reply: model unavailable" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestSendMessageReturnsMissingConversationError(t *testing.T) {
	t.Parallel()

	service := NewService(newInMemoryRepository(), fixedResponder{reply: "unused"})

	_, err := service.SendMessage(context.Background(), "conv-404", "hello")
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}
