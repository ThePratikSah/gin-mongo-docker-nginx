// Package handler is the HTTP transport layer. It translates HTTP <-> domain
// calls and depends only on the small service interface it declares.
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ginmongo/internal/domain"
)

// UserService is the behaviour the handler needs from the service layer.
type UserService interface {
	Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error)
	Get(ctx context.Context, id string) (domain.User, error)
	List(ctx context.Context, limit, offset int) ([]domain.User, error)
	Update(ctx context.Context, id string, in domain.UpdateUserInput) (domain.User, error)
	Delete(ctx context.Context, id string) error
}

type UserHandler struct {
	svc UserService
}

func NewUserHandler(svc UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Register mounts the user routes. It satisfies router.Registrar.
func (h *UserHandler) Register(r gin.IRouter) {
	g := r.Group("/api/v1/users")
	g.POST("", h.create)
	g.GET("", h.list)
	g.GET("/:id", h.get)
	g.PATCH("/:id", h.update)
	g.DELETE("/:id", h.delete)
}

// Request DTOs: transport-level validation lives here via Gin's binding tags.
type createUserRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=100"`
	Email string `json:"email" binding:"required,email,max=254"`
}

type updateUserRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=1,max=100"`
	Email *string `json:"email" binding:"omitempty,email,max=254"`
}

type listQuery struct {
	Limit  int `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset" binding:"omitempty,min=0"`
}

func (h *UserHandler) create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	u, err := h.svc.Create(c.Request.Context(), domain.CreateUserInput{Name: req.Name, Email: req.Email})
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Location", "/api/v1/users/"+u.ID)
	c.JSON(http.StatusCreated, u)
}

func (h *UserHandler) get(c *gin.Context) {
	u, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) list(c *gin.Context) {
	var q listQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		badRequest(c, err)
		return
	}

	users, err := h.svc.List(c.Request.Context(), q.Limit, q.Offset)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "limit": q.Limit, "offset": q.Offset})
}

func (h *UserHandler) update(c *gin.Context) {
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	u, err := h.svc.Update(c.Request.Context(), c.Param("id"), domain.UpdateUserInput{Name: req.Name, Email: req.Email})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
