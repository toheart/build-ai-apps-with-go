package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	appchat "github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/application/chat"
	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/domain/conversation"
	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/interfaces/http/response"
)

const (
	codeConversationInvalid = 200001
	codeConversationMissing = 200002
	codeConversationCreate  = 200003
	codeConversationSend    = 200004
	codeConversationGet     = 200005
)

type createConversationResponse struct {
	ConversationID string `json:"conversationId"`
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

type conversationMessageResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type conversationResponse struct {
	ID       string                        `json:"id"`
	Messages []conversationMessageResponse `json:"messages"`
}

type ConversationHandler struct {
	service *appchat.Service
}

func NewConversationHandler(service *appchat.Service) *ConversationHandler {
	return &ConversationHandler{service: service}
}

func (h *ConversationHandler) RegisterRoutes(router *gin.RouterGroup) {
	conversations := router.Group("/conversations")
	conversations.POST("", h.CreateConversation)
	conversations.GET("/:id", h.GetConversation)
	conversations.POST("/:id/messages", h.SendMessage)
}

// CreateConversation godoc
//
//	@Summary		Create a conversation
//	@Tags			conversations
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Envelope
//	@Failure		500	{object}	response.Envelope
//	@Router			/conversations [post]
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	conv, err := h.service.CreateConversation(c.Request.Context())
	if err != nil {
		response.Error(
			c,
			http.StatusInternalServerError,
			codeConversationCreate,
			"failed to create conversation",
		)
		return
	}

	response.Success(c, createConversationResponse{
		ConversationID: conv.ID(),
	})
}

// GetConversation godoc
//
//	@Summary		Get conversation details
//	@Tags			conversations
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Conversation ID"
//	@Success		200	{object}	response.Envelope
//	@Failure		400	{object}	response.Envelope
//	@Failure		404	{object}	response.Envelope
//	@Failure		500	{object}	response.Envelope
//	@Router			/conversations/{id} [get]
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	conv, err := h.service.GetConversation(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeConversationError(c, err, codeConversationGet, "failed to get conversation")
		return
	}

	response.Success(c, toConversationResponse(conv))
}

// SendMessage godoc
//
//	@Summary		Send a conversation message
//	@Tags			conversations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Conversation ID"
//	@Param			request	body		sendMessageRequest	true	"Message payload"
//	@Success		200		{object}	response.Envelope
//	@Failure		400		{object}	response.Envelope
//	@Failure		404		{object}	response.Envelope
//	@Failure		500		{object}	response.Envelope
//	@Router			/conversations/{id}/messages [post]
func (h *ConversationHandler) SendMessage(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			codeConversationInvalid,
			"invalid request body",
			err.Error(),
		)
		return
	}

	conv, err := h.service.SendMessage(c.Request.Context(), c.Param("id"), req.Content)
	if err != nil {
		h.writeConversationError(c, err, codeConversationSend, "failed to send message")
		return
	}

	response.Success(c, toConversationResponse(conv))
}

func (h *ConversationHandler) writeConversationError(
	c *gin.Context,
	err error,
	fallbackCode int,
	fallbackMessage string,
) {
	switch {
	case errors.Is(err, conversation.ErrEmptyConversationID):
		response.Error(c, http.StatusBadRequest, codeConversationInvalid, err.Error())
	case errors.Is(err, conversation.ErrEmptyMessageContent):
		response.Error(c, http.StatusBadRequest, codeConversationInvalid, err.Error())
	case errors.Is(err, appchat.ErrConversationNotFound):
		response.Error(c, http.StatusNotFound, codeConversationMissing, "conversation not found")
	default:
		response.Error(c, http.StatusInternalServerError, fallbackCode, fallbackMessage)
	}
}

func toConversationResponse(conv conversation.Conversation) conversationResponse {
	messages := conv.Messages()
	items := make([]conversationMessageResponse, 0, len(messages))

	for _, item := range messages {
		items = append(items, conversationMessageResponse{
			Role:    string(item.Role),
			Content: item.Content,
		})
	}

	return conversationResponse{
		ID:       conv.ID(),
		Messages: items,
	}
}
