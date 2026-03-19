package conversation

import (
	"errors"
	"testing"
)

func TestNewConversationRequiresID(t *testing.T) {
	t.Parallel()

	_, err := New("   ")
	if !errors.Is(err, ErrEmptyConversationID) {
		t.Fatalf("expected ErrEmptyConversationID, got %v", err)
	}
}

func TestConversationAddsTrimmedMessages(t *testing.T) {
	t.Parallel()

	conv, err := New("conv-001")
	if err != nil {
		t.Fatalf("new conversation: %v", err)
	}

	if err := conv.AddUserMessage("  hello  "); err != nil {
		t.Fatalf("add user message: %v", err)
	}

	if err := conv.AddAssistantMessage("  hi there "); err != nil {
		t.Fatalf("add assistant message: %v", err)
	}

	got := conv.Messages()
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}

	if got[0].Role != RoleUser || got[0].Content != "hello" {
		t.Fatalf("unexpected first message: %+v", got[0])
	}

	if got[1].Role != RoleAssistant || got[1].Content != "hi there" {
		t.Fatalf("unexpected second message: %+v", got[1])
	}
}

func TestConversationRejectsEmptyMessages(t *testing.T) {
	t.Parallel()

	conv, err := New("conv-001")
	if err != nil {
		t.Fatalf("new conversation: %v", err)
	}

	if err := conv.AddUserMessage("   "); !errors.Is(err, ErrEmptyMessageContent) {
		t.Fatalf("expected ErrEmptyMessageContent, got %v", err)
	}
}

func TestConversationMessagesReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	conv, err := Restore("conv-001", []Message{
		{Role: RoleUser, Content: "hello"},
	})
	if err != nil {
		t.Fatalf("restore conversation: %v", err)
	}

	messages := conv.Messages()
	messages[0].Content = "changed"

	freshMessages := conv.Messages()
	if freshMessages[0].Content != "hello" {
		t.Fatalf("expected defensive copy, got %q", freshMessages[0].Content)
	}
}
