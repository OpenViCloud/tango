package domain

import "context"

type UserListOptions struct {
	PageIndex  int
	PageSize   int
	SearchText string
	OrderBy    string
	Ascending  bool
}

type UserListResult struct {
	Items      []*User
	TotalItems int64
}

type UserRepository interface {
	Save(ctx context.Context, user *User) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetAll(ctx context.Context, opts UserListOptions) (*UserListResult, error)
	HasAnyUser(ctx context.Context) (bool, error)
}
