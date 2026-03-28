package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrProjectNotFound     = errors.New("project not found")
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrResourceNotFound   = errors.New("resource not found")
	ErrResourceNotStarted = errors.New("resource container has not been created yet")
)

// ErrHostPortConflict is returned when a host port is already occupied by
// another running resource.
type ErrHostPortConflict struct {
	Port         int
	OccupiedByID   string
	OccupiedByName string
}

func (e *ErrHostPortConflict) Error() string {
	return fmt.Sprintf("host port %d is already in use by resource %q", e.Port, e.OccupiedByName)
}

// UserFacingError is an error whose message is safe to return directly to the
// API caller as a 400 Bad Request (e.g. port conflict, invalid image).
type UserFacingError struct{ msg string }

func NewUserFacingError(msg string) *UserFacingError { return &UserFacingError{msg: msg} }
func (e *UserFacingError) Error() string             { return e.msg }

// IsUserFacing reports whether err (or any in its chain) is a UserFacingError.
func IsUserFacing(err error) bool {
	var target *UserFacingError
	return errors.As(err, &target)
}

type Project struct {
	ID           string
	Name         string
	Description  string
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Environments []Environment
}

type CreateProjectInput struct {
	ID          string
	Name        string
	Description string
	CreatedBy   string
}

type ProjectRepository interface {
	Create(ctx context.Context, input CreateProjectInput) (*Project, error)
	List(ctx context.Context) ([]*Project, error)
	GetByID(ctx context.Context, id string) (*Project, error)
	Update(ctx context.Context, id, name, description string) (*Project, error)
	Delete(ctx context.Context, id string) error
}
