package metadata

import (
	"context"
	"fmt"
	"strconv"

	"github.com/luevano/libmangal/logger"
	"github.com/philippgille/gokv"
	"golang.org/x/mod/semver"
)

var _ Provider = (*ProviderWithCache)(nil)

// ProviderInfo is the passport of the metadata provider.
type ProviderInfo struct {
	// ID is the unique identifier of the provider.
	//
	// For a ProviderWithCache this is used as the cache DB name.
	ID string `json:"id"`

	// Name is the non-empty name of the provider.
	Name string `json:"name"`

	// Version is a semantic version of the provider.
	//
	// "v" prefix is not permitted.
	// E.g. "0.1.0" is valid, but "v0.1.0" is not.
	//
	// See https://semver.org/
	Version string `json:"version"`

	// Description of the provider. May be empty.
	Description string `json:"description"`

	// Website of the provider. May be empty.
	Website string `json:"website"`
}

// Validate checks if the ProviderInfo is valid.
//
// This means that ID and ID are non-empty and
// Version is a valid semver.
func (p ProviderInfo) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("ID must be non-empty")
	}

	if p.Name == "" {
		return fmt.Errorf("Name must be non-empty")
	}

	// according to the semver specification,
	// versions should not have "v" prefix. E.g. v0.1.0 isn't a valid semver,
	// however, for some bizarre reason, Go semver package requires this prefix.
	if !semver.IsValid("v" + p.Version) {
		return fmt.Errorf("invalid semver: %s", p.Version)
	}

	return nil
}

// Provider exposes methods for searching mangas, getting chapters, pages and images.
type Provider interface {
	fmt.Stringer

	// Info information about Provider.
	Info() ProviderInfo

	// SetLogger sets logger to use for this provider.
	//
	// Setting a nil logger will create a new one.
	SetLogger(*logger.Logger)

	// Logger returns the set logger.
	//
	// Always returns a non-nil logger.
	Logger() *logger.Logger

	// SearchByID for metadata with the given id.
	//
	// Implementation should only handle the request and and marshaling.
	SearchByID(ctx context.Context, id int) (Metadata, bool, error)

	// Search for metadata with the given query.
	//
	// Implementation should only handle the request and and marshaling.
	Search(ctx context.Context, query string) ([]Metadata, error)
}

// ProviderWithCache is a Provider implementation with cache features.
//
// This is a wrapper on a normal Provider.
type ProviderWithCache struct {
	provider Provider
	store    store
	logger   *logger.Logger
}

// NewProviderWithCache constructs new Provider with cache given the Provider.
func NewProviderWithCache(
	provider Provider,
	cacheStore func(dbName, bucketName string) (gokv.Store, error),
) (*ProviderWithCache, error) {
	if provider == nil {
		return nil, Error("nil Provider passed to ProviderWithCache")
	}

	s := store{
		openStore: func(bucketName string) (gokv.Store, error) {
			return cacheStore(provider.Info().ID, bucketName)
		},
	}

	// ensure the logger is non-nil
	l := provider.Logger()
	if l == nil {
		l = logger.NewLogger()
	}

	p := &ProviderWithCache{
		provider: provider,
		store:    s,
		logger:   l,
	}

	return p, nil
}

func (p *ProviderWithCache) String() string {
	return p.provider.String()
}

// Info information about Provider.
func (p *ProviderWithCache) Info() ProviderInfo {
	return p.provider.Info()
}

// SetLogger sets logger to use for this provider.
//
// Setting a nil logger will create a new one.
func (p *ProviderWithCache) SetLogger(_logger *logger.Logger) {
	if _logger != nil {
		// p.logger is guaranteed to be non-nil
		*p.logger = *_logger
	} else {
		p.logger = logger.NewLogger()
	}
	p.provider.SetLogger(p.logger)
}

// Logger returns the set logger.
//
// Always returns a non-nil logger.
func (p *ProviderWithCache) Logger() *logger.Logger {
	return p.logger
}

// SearchByID for metadata with the given id.
//
// Implementation should only handle the request and and marshaling.
func (p *ProviderWithCache) SearchByID(ctx context.Context, id int) (Metadata, bool, error) {
	p.logger.Log("searching manga metadata with id %d on %q", id, p.provider.Info().ID)
	meta, found, err := p.store.getMeta(id)
	if err != nil {
		return nil, false, Error(err.Error())
	}
	if found {
		return meta, true, nil
	}

	meta, ok, err := p.provider.SearchByID(ctx, id)
	if err != nil {
		return nil, false, Error(err.Error())
	}
	if !ok {
		return nil, false, nil
	}

	err = p.store.setMeta(id, meta)
	if err != nil {
		return nil, false, Error(err.Error())
	}

	return meta, true, nil
}

// Search for metadata with the given query.
//
// Implementation should only handle the request and and marshaling.
func (p *ProviderWithCache) Search(ctx context.Context, query string) ([]Metadata, error) {
	p.logger.Log("searching manga metadata with query %q on %q", query, p.provider.Info().ID)
	ids, found, err := p.store.getQueryIDs(query)
	if err != nil {
		return nil, Error(err.Error())
	}
	if found {
		var metas []Metadata
		for _, id := range ids {
			meta, ok, err := p.SearchByID(ctx, id)
			if err != nil {
				return nil, err
			}
			if ok {
				metas = append(metas, meta)
			}
		}
		return metas, nil
	}

	metas, err := p.provider.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	ids = make([]int, len(metas))
	for i, meta := range metas {
		id, err := strconv.Atoi(meta.ID().Raw)
		if err != nil {
			return nil, Error(err.Error())
		}
		err = p.store.setMeta(id, meta)
		if err != nil {
			return nil, Error(err.Error())
		}

		ids[i] = id
	}

	err = p.store.setQueryIDs(query, ids)
	if err != nil {
		return nil, Error(err.Error())
	}

	return metas, nil
}