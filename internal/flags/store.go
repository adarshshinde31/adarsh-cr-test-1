package flags

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("feature flag not found")
	ErrExists   = errors.New("feature flag already exists")
	ErrInvalid  = errors.New("invalid feature flag")
)

// Store is the persistence contract for feature flags. Implementations may be
// backed by the local filesystem, a database, a remote config service, etc.
type Store interface {
	Create(ctx context.Context, flag FeatureFlag) error
	Get(ctx context.Context, key string) (FeatureFlag, error)
	List(ctx context.Context) ([]FeatureFlag, error)
	Update(ctx context.Context, flag FeatureFlag) error
	Delete(ctx context.Context, key string) error
}
