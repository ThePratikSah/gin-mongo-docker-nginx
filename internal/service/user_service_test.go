package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"ginmongo/internal/domain"
)

// fakeRepo is a hand-written test double. No mocking library, no database.
type fakeRepo struct {
	created []domain.User
	err     error
}

func (f *fakeRepo) Create(_ context.Context, u domain.User) (domain.User, error) {
	if f.err != nil {
		return domain.User{}, f.err
	}
	u.ID = "507f1f77bcf86cd799439011"
	f.created = append(f.created, u)
	return u, nil
}
func (f *fakeRepo) GetByID(context.Context, string) (domain.User, error) { return domain.User{}, nil }
func (f *fakeRepo) List(context.Context, int, int) ([]domain.User, error) { return nil, nil }
func (f *fakeRepo) Update(context.Context, string, domain.UserUpdate) (domain.User, error) {
	return domain.User{}, nil
}
func (f *fakeRepo) Delete(context.Context, string) error { return nil }

func newTestService(repo UserRepository, now time.Time) *UserService {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewUserService(repo, log, func() time.Time { return now })
}

func TestCreate_NormalizesInput(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := &fakeRepo{}
	svc := newTestService(repo, now)

	u, err := svc.Create(context.Background(), domain.CreateUserInput{
		Name:  "  Ada ",
		Email: " ADA@Example.com ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "Ada" || u.Email != "ada@example.com" {
		t.Fatalf("input not normalised: %+v", u)
	}
	if !u.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %v, want %v", u.CreatedAt, now)
	}
}

func TestCreate_RejectsEmptyName(t *testing.T) {
	repo := &fakeRepo{}
	svc := newTestService(repo, time.Now())

	_, err := svc.Create(context.Background(), domain.CreateUserInput{Name: "   ", Email: "a@b.co"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if len(repo.created) != 0 {
		t.Fatal("repository must not be called on invalid input")
	}
}

func TestCreate_PropagatesDuplicateEmail(t *testing.T) {
	svc := newTestService(&fakeRepo{err: domain.ErrEmailTaken}, time.Now())

	_, err := svc.Create(context.Background(), domain.CreateUserInput{Name: "Ada", Email: "a@b.co"})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}
