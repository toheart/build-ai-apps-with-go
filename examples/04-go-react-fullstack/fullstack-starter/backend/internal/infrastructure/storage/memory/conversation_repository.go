package memory

import (
	"context"
	"sync"

	appchat "github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/application/chat"
	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/domain/conversation"
)

var _ appchat.Repository = (*ConversationRepository)(nil)

type ConversationRepository struct {
	mu    sync.RWMutex
	items map[string]conversation.Conversation
}

func NewConversationRepository() *ConversationRepository {
	return &ConversationRepository{
		items: make(map[string]conversation.Conversation),
	}
}

func (r *ConversationRepository) Create(
	ctx context.Context,
	conv conversation.Conversation,
) error {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[conv.ID()]; exists {
		return appchat.ErrConversationExists
	}

	cloned, err := cloneConversation(conv)
	if err != nil {
		return err
	}

	r.items[conv.ID()] = cloned
	return nil
}

func (r *ConversationRepository) Load(
	ctx context.Context,
	id string,
) (conversation.Conversation, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	conv, exists := r.items[id]
	if !exists {
		return conversation.Conversation{}, appchat.ErrConversationNotFound
	}

	return cloneConversation(conv)
}

func (r *ConversationRepository) Save(
	ctx context.Context,
	conv conversation.Conversation,
) error {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[conv.ID()]; !exists {
		return appchat.ErrConversationNotFound
	}

	cloned, err := cloneConversation(conv)
	if err != nil {
		return err
	}

	r.items[conv.ID()] = cloned
	return nil
}

func cloneConversation(conv conversation.Conversation) (conversation.Conversation, error) {
	return conversation.Restore(conv.ID(), conv.Messages())
}
