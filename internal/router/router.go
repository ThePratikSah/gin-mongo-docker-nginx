// Package router builds the Gin engine from already-constructed handlers.
package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ginmongo/internal/middleware"
)

// Registrar is implemented by every handler that owns routes.
type Registrar interface {
	Register(r gin.IRouter)
}

type Options struct {
	Production     bool
	TrustedProxies []string
}

func New(opts Options, log *slog.Logger, registrars ...Registrar) (*gin.Engine, error) {
	if opts.Production {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(opts.TrustedProxies); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}

	r.Use(
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.Recovery(log),
	)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
	})

	for _, reg := range registrars {
		reg.Register(r)
	}
	return r, nil
}
