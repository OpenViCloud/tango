package command

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"tango/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type CreateUserCommand struct {
	ID        string
	Email     string
	Nickname  string
	FirstName string
	LastName  string
	Phone     string
	Address   string
	Password  string
}

type CreateUserHandler struct {
	repo domain.UserRepository
}

func NewCreateUserHandler(repo domain.UserRepository) *CreateUserHandler {
	return &CreateUserHandler{repo: repo}
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (*domain.User, error) {
	existing, err := h.repo.FindByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(cmd.ID, cmd.Email, cmd.Nickname, cmd.FirstName, cmd.LastName, cmd.Phone, cmd.Address, string(passwordHash))
	if err != nil {
		return nil, err
	}

	saved, err := h.repo.Save(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}
	return saved, nil
}

type UpdateUserCommand struct {
	ID        string
	Nickname  string
	FirstName string
	LastName  string
	Phone     string
	Address   string
}

type UpdateUserHandler struct {
	repo domain.UserRepository
}

func NewUpdateUserHandler(repo domain.UserRepository) *UpdateUserHandler {
	return &UpdateUserHandler{repo: repo}
}

func (h *UpdateUserHandler) Handle(ctx context.Context, cmd UpdateUserCommand) (*domain.User, error) {
	user, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if err := user.Rename(cmd.FirstName, cmd.LastName); err != nil {
		return nil, err
	}
	user.UpdateNickname(cmd.Nickname)
	user.UpdatePhone(cmd.Phone)
	user.UpdateAddress(cmd.Address)
	updated, err := h.repo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return updated, nil
}

type ChangePasswordCommand struct {
	ID              string
	CurrentPassword string
	NewPassword     string
}

type ChangePasswordHandler struct {
	repo domain.UserRepository
}

func NewChangePasswordHandler(repo domain.UserRepository) *ChangePasswordHandler {
	return &ChangePasswordHandler{repo: repo}
}

func (h *ChangePasswordHandler) Handle(ctx context.Context, cmd ChangePasswordCommand) error {
	if cmd.ID == "" || cmd.CurrentPassword == "" || len(cmd.NewPassword) < 6 {
		return domain.ErrInvalidInput
	}

	user, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cmd.CurrentPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cmd.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := user.ChangePassword(string(passwordHash)); err != nil {
		return err
	}

	if _, err := h.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	return nil
}

type BanUserCommand struct {
	ID string
}

type BanUserHandler struct {
	repo domain.UserRepository
}

func NewBanUserHandler(repo domain.UserRepository) *BanUserHandler {
	return &BanUserHandler{repo: repo}
}

func (h *BanUserHandler) Handle(ctx context.Context, cmd BanUserCommand) error {
	user, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if err := user.Ban(); err != nil {
		return err
	}
	_, err = h.repo.Update(ctx, user)
	return err
}

type DeleteUserCommand struct {
	ID string
}

type DeleteUserHandler struct {
	repo domain.UserRepository
}

func NewDeleteUserHandler(repo domain.UserRepository) *DeleteUserHandler {
	return &DeleteUserHandler{repo: repo}
}

func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
	if _, err := h.repo.GetByID(ctx, cmd.ID); err != nil {
		return err
	}
	return h.repo.Delete(ctx, cmd.ID)
}

type RegisterUserCommand struct {
	Email     string
	FirstName string
	LastName  string
	Password  string
}

type RegisterUserHandler struct {
	userRepo domain.UserRepository
	roleRepo domain.RoleRepository
}

func NewRegisterUserHandler(userRepo domain.UserRepository, roleRepo domain.RoleRepository) *RegisterUserHandler {
	return &RegisterUserHandler{userRepo: userRepo, roleRepo: roleRepo}
}

func (h *RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) (*domain.User, error) {
	has, err := h.userRepo.HasAnyUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("check existing users: %w", err)
	}
	if has {
		return nil, domain.ErrRegistrationClosed
	}

	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}
	id := "user_" + hex.EncodeToString(idBytes)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(id, cmd.Email, cmd.Email, cmd.FirstName, cmd.LastName, "", "", string(passwordHash))
	if err != nil {
		return nil, err
	}

	saved, err := h.userRepo.Save(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	adminRole, err := h.roleRepo.GetByName(ctx, "admin")
	if err != nil {
		return nil, fmt.Errorf("get admin role: %w", err)
	}
	if err := h.roleRepo.AssignRoleToUser(ctx, saved.ID, adminRole.ID); err != nil {
		return nil, fmt.Errorf("assign admin role: %w", err)
	}

	return saved, nil
}
