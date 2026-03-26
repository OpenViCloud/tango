package domain

import "context"

type ChannelListOptions struct {
	PageIndex  int
	PageSize   int
	SearchText string
	OrderBy    string
	Ascending  bool
}

type ChannelListResult struct {
	Items      []*Channel
	TotalItems int64
}

type ChannelRepository interface {
	Save(ctx context.Context, channel *Channel) (*Channel, error)
	Update(ctx context.Context, channel *Channel) (*Channel, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Channel, error)
	GetByName(ctx context.Context, name string) (*Channel, error)
	GetAll(ctx context.Context, opts ChannelListOptions) (*ChannelListResult, error)
	ListByWorkspaceID(ctx context.Context, workspaceID string) ([]*Channel, error)
	ListActiveConfigured(ctx context.Context) ([]*Channel, error)
	SetStatusByKindExcept(ctx context.Context, kind ChannelKind, exceptID string, status ChannelStatus) error
}
