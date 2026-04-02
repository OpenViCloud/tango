package command

import (
	"context"
	"testing"
	"time"

	"tango/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepository struct {
	users map[string]*domain.User
}

func newFakeUserRepository(users ...*domain.User) *fakeUserRepository {
	repo := &fakeUserRepository{
		users: make(map[string]*domain.User, len(users)),
	}
	for _, user := range users {
		repo.users[user.ID] = cloneUser(user)
	}
	return repo
}

func (r *fakeUserRepository) Save(_ context.Context, user *domain.User) (*domain.User, error) {
	r.users[user.ID] = cloneUser(user)
	return cloneUser(user), nil
}

func (r *fakeUserRepository) Update(_ context.Context, user *domain.User) (*domain.User, error) {
	if _, ok := r.users[user.ID]; !ok {
		return nil, domain.ErrUserNotFound
	}
	r.users[user.ID] = cloneUser(user)
	return cloneUser(user), nil
}

func (r *fakeUserRepository) Delete(_ context.Context, id string) error {
	delete(r.users, id)
	return nil
}

func (r *fakeUserRepository) FindByID(_ context.Context, id string) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	return cloneUser(user), nil
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return cloneUser(user), nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, id string) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return cloneUser(user), nil
}

func (r *fakeUserRepository) GetAll(_ context.Context, _ domain.UserListOptions) (*domain.UserListResult, error) {
	return &domain.UserListResult{}, nil
}

func cloneUser(user *domain.User) *domain.User {
	if user == nil {
		return nil
	}
	copy := *user
	return &copy
}

func TestChangePasswordHandlerUpdatesPasswordHash(t *testing.T) {
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate old hash: %v", err)
	}

	user := &domain.User{
		ID:           "user-1",
		Email:        "admin@example.com",
		FirstName:    "Admin",
		LastName:     "User",
		PasswordHash: string(oldHash),
		Status:       domain.UserStatusActive,
		CreatedAt:    time.Now().UTC().Add(-time.Hour),
		UpdatedAt:    time.Now().UTC().Add(-time.Hour),
	}
	repo := newFakeUserRepository(user)
	handler := NewChangePasswordHandler(repo)

	err = handler.Handle(context.Background(), ChangePasswordCommand{
		ID:              user.ID,
		CurrentPassword: "old-password",
		NewPassword:     "new-password",
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("new-password")) != nil {
		t.Fatal("expected updated password hash to match new password")
	}
	if bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("old-password")) == nil {
		t.Fatal("expected old password to no longer match stored hash")
	}
}

func TestChangePasswordHandlerRejectsInvalidCurrentPassword(t *testing.T) {
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate old hash: %v", err)
	}

	user := &domain.User{
		ID:           "user-1",
		Email:        "admin@example.com",
		FirstName:    "Admin",
		LastName:     "User",
		PasswordHash: string(oldHash),
		Status:       domain.UserStatusActive,
		CreatedAt:    time.Now().UTC().Add(-time.Hour),
		UpdatedAt:    time.Now().UTC().Add(-time.Hour),
	}
	repo := newFakeUserRepository(user)
	handler := NewChangePasswordHandler(repo)

	err = handler.Handle(context.Background(), ChangePasswordCommand{
		ID:              user.ID,
		CurrentPassword: "wrong-password",
		NewPassword:     "new-password",
	})
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	stored, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get stored user: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("old-password")) != nil {
		t.Fatal("expected original password hash to remain unchanged")
	}
}
