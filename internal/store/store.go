package store

import (
	"context"
	"errors"
	"github.com/Arjunren/asset-management-system/internal/domain"
	"time"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrConflict    = errors.New("conflict")
	ErrUnavailable = errors.New("asset unavailable")
)

type Store interface {
	UserByEmail(context.Context, string) (domain.User, error)
	UserByID(context.Context, int64) (domain.User, error)
	CreateUser(context.Context, domain.User) (domain.User, error)
	CreateSessionToken(context.Context, string, int64, time.Time) error
	SessionActive(context.Context, string, int64) (bool, error)
	RevokeSession(context.Context, string, int64) error
	CreateAsset(context.Context, domain.Asset) (domain.Asset, error)
	UpdateAsset(context.Context, domain.Asset) (domain.Asset, error)
	AssetByID(context.Context, int64) (domain.Asset, error)
	ListAssets(context.Context, domain.ListParams) ([]domain.Asset, int64, error)
	TransitionAsset(context.Context, int64, domain.AssetStatus, *int64, string, int64) (domain.Asset, error)
	ListEvents(context.Context, int64) ([]domain.AssetEvent, error)
	Dashboard(context.Context, int64, domain.Role) (domain.Dashboard, error)
}
