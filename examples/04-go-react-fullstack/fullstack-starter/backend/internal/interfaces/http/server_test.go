package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	appchat "github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/application/chat"
	appsample "github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/application/sample"
	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/infrastructure/storage/memory"
	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/interfaces/http/handler"

	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/conf"
)

type envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Detail  any    `json:"detail,omitempty"`
}

type samplePayload struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Updated  string `json:"updatedAt"`
}

type createConversationPayload struct {
	ConversationID string `json:"conversationId"`
}

type conversationMessagePayload struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type conversationPayload struct {
	ID       string                       `json:"id"`
	Messages []conversationMessagePayload `json:"messages"`
}

func TestServerRoutes(t *testing.T) {
	t.Parallel()

	server := buildTestServer()

	t.Run("healthz uses the shared response envelope", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		recorder := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", recorder.Code)
		}

		var payload envelope[map[string]string]
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal healthz response: %v", err)
		}

		if payload.Code != 0 || payload.Message != "success" {
			t.Fatalf("unexpected envelope: %+v", payload)
		}

		if payload.Data["status"] != "ok" {
			t.Fatalf("expected health status ok, got %q", payload.Data["status"])
		}
	})

	t.Run("sample route returns starter sample items", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/api/v1/samples", nil)
		recorder := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", recorder.Code)
		}

		var payload envelope[[]samplePayload]
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal sample response: %v", err)
		}

		if payload.Code != 0 || payload.Message != "success" {
			t.Fatalf("unexpected envelope: %+v", payload)
		}

		if len(payload.Data) != 3 {
			t.Fatalf("expected 3 sample items, got %d", len(payload.Data))
		}
	})

	t.Run("conversation routes create send and fetch history", func(t *testing.T) {
		t.Parallel()

		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", nil)
		createRecorder := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(createRecorder, createReq)

		if createRecorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", createRecorder.Code)
		}

		var createPayload envelope[createConversationPayload]
		if err := json.Unmarshal(createRecorder.Body.Bytes(), &createPayload); err != nil {
			t.Fatalf("unmarshal create response: %v", err)
		}

		if createPayload.Code != 0 || createPayload.Data.ConversationID == "" {
			t.Fatalf("unexpected create payload: %+v", createPayload)
		}

		sendBody := bytes.NewBufferString(`{"content":"  hello backend  "}`)
		sendReq := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/conversations/"+createPayload.Data.ConversationID+"/messages",
			sendBody,
		)
		sendReq.Header.Set("Content-Type", "application/json")
		sendRecorder := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(sendRecorder, sendReq)

		if sendRecorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", sendRecorder.Code)
		}

		var sendPayload envelope[conversationPayload]
		if err := json.Unmarshal(sendRecorder.Body.Bytes(), &sendPayload); err != nil {
			t.Fatalf("unmarshal send response: %v", err)
		}

		if len(sendPayload.Data.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(sendPayload.Data.Messages))
		}

		if sendPayload.Data.Messages[0].Content != "hello backend" {
			t.Fatalf("expected trimmed user message, got %q", sendPayload.Data.Messages[0].Content)
		}

		if sendPayload.Data.Messages[1].Content != "已收到：hello backend" {
			t.Fatalf(
				"expected assistant reply, got %q",
				sendPayload.Data.Messages[1].Content,
			)
		}

		getReq := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/conversations/"+createPayload.Data.ConversationID,
			nil,
		)
		getRecorder := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(getRecorder, getReq)

		if getRecorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", getRecorder.Code)
		}

		var getPayload envelope[conversationPayload]
		if err := json.Unmarshal(getRecorder.Body.Bytes(), &getPayload); err != nil {
			t.Fatalf("unmarshal get response: %v", err)
		}

		if getPayload.Data.ID != createPayload.Data.ConversationID {
			t.Fatalf("expected id %q, got %q", createPayload.Data.ConversationID, getPayload.Data.ID)
		}

		if len(getPayload.Data.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(getPayload.Data.Messages))
		}
	})

	t.Run("conversation routes return not found for missing conversation", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/conv-missing", nil)
		recorder := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", recorder.Code)
		}

		var payload envelope[map[string]any]
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal not found response: %v", err)
		}

		if payload.Code != 200002 {
			t.Fatalf("expected business code 200002, got %d", payload.Code)
		}
	})

	t.Run("conversation routes reject invalid message content", func(t *testing.T) {
		t.Parallel()

		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", nil)
		createRecorder := httptest.NewRecorder()
		server.server.Handler.ServeHTTP(createRecorder, createReq)

		var createPayload envelope[createConversationPayload]
		if err := json.Unmarshal(createRecorder.Body.Bytes(), &createPayload); err != nil {
			t.Fatalf("unmarshal create response: %v", err)
		}

		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/conversations/"+createPayload.Data.ConversationID+"/messages",
			bytes.NewBufferString(`{"content":"   "}`),
		)
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", recorder.Code)
		}

		var payload envelope[map[string]any]
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal validation response: %v", err)
		}

		if payload.Code != 200001 {
			t.Fatalf("expected business code 200001, got %d", payload.Code)
		}
	})
}

func buildTestServer() *Server {
	cfg := conf.Config{
		App: conf.AppConfig{
			Name:    "fullstack-starter",
			RunMode: "test",
		},
		HTTP: conf.HTTPConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sampleRepo := memory.NewSampleRepository()
	sampleService := appsample.NewService(sampleRepo)
	sampleHandler := handler.NewSampleHandler(sampleService)

	conversationRepo := memory.NewConversationRepository()
	chatService := appchat.NewService(
		conversationRepo,
		memory.NewRuleBasedResponder(),
	)
	conversationHandler := handler.NewConversationHandler(chatService)

	return New(cfg, logger, sampleHandler, conversationHandler)
}
