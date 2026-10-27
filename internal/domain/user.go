// Package domain holds business types and sentinel errors.
// It imports nothing from the rest of the app and knows nothing about
// Mongo or Gin.
package domain

import (
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidID    = errors.New("invalid id")
	ErrEmailTaken   = errors.New("email already in use")
	ErrInvalidInput = errors.New("invalid input")
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserInput is what the service layer accepts for creation.
type CreateUserInput struct {
	Name  string
	Email string
}

// UpdateUserInput is a partial update; nil fields are left untouched.
type UpdateUserInput struct {
	Name  *string
	Email *string
}

// UserUpdate is the normalised patch the repository applies.
type UserUpdate struct {
	Name      *string
	Email     *string
	UpdatedAt time.Time
}
