package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ginmongo/internal/domain"
)

type errorResponse struct {
	Error string `json:"error"`
}

// writeError maps domain errors to HTTP responses. Unknown errors are attached
// to the Gin context (so the logger middleware records them) and hidden from
// the client.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, errorResponse{err.Error()})
	case errors.Is(err, domain.ErrInvalidID):
		c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse{err.Error()})
	case errors.Is(err, domain.ErrEmailTaken):
		c.AbortWithStatusJSON(http.StatusConflict, errorResponse{err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, errorResponse{err.Error()})
	default:
		_ = c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse{"internal server error"})
	}
}

func badRequest(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse{err.Error()})
}
