// Package service holds business logic.
// It declares the repository interface it needs (consumer-side interface),
// so it depends on behaviour, not on MongoDB.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"ginmongo/internal/domain"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// UserRepository is the persistence behaviour UserService requires.
// *repository.UserRepository satisfies it implicitly.
type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	List(ctx context.Context, limit, offset int) ([]domain.User, error)
	Update(ctx context.Context, id string, upd domain.UserUpdate) (domain.User, error)
	Delete(ctx context.Context, id string) error
}

// Clock is injected so tests can control time.
type Clock func() time.Time

type UserService struct {
	repo UserRepository
	log  *slog.Logger
	now  Clock
}

func NewUserService(repo UserRepository, log *slog.Logger, now Clock) *UserService {
	return &UserService{repo: repo, log: log, now: now}
}

func (s *UserService) Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error) {
	name := strings.TrimSpace(in.Name)
	email := normalizeEmail(in.Email)

	if name == "" {
		return domain.User{}, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	if email == "" {
		return domain.User{}, fmt.Errorf("%w: email is required", domain.ErrInvalidInput)
	}

	now := s.now().UTC()
	u, err := s.repo.Create(ctx, domain.User{
		Name:      name,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return domain.User{}, err
	}

	s.log.InfoContext(ctx, "user created", "user_id", u.ID)
	return u, nil
}

func (s *UserService) Get(ctx context.Context, id string) (domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *UserService) Update(ctx context.Context, id string, in domain.UpdateUserInput) (domain.User, error) {
	if in.Name == nil && in.Email == nil {
		return domain.User{}, fmt.Errorf("%w: nothing to update", domain.ErrInvalidInput)
	}

	upd := domain.UserUpdate{UpdatedAt: s.now().UTC()}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return domain.User{}, fmt.Errorf("%w: name cannot be empty", domain.ErrInvalidInput)
		}
		upd.Name = &name
	}
	if in.Email != nil {
		email := normalizeEmail(*in.Email)
		if email == "" {
			return domain.User{}, fmt.Errorf("%w: email cannot be empty", domain.ErrInvalidInput)
		}
		upd.Email = &email
	}

	return s.repo.Update(ctx, id, upd)
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "user deleted", "user_id", id)
	return nil
}

func normalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}
