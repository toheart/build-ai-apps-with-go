package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Detail  any    `json:"detail,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, statusCode, businessCode int, message string, detail ...any) {
	body := Envelope{
		Code:    businessCode,
		Message: message,
	}
	if len(detail) > 0 {
		body.Detail = detail[0]
	}

	c.JSON(statusCode, body)
}
