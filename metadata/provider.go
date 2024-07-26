package metadata

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/luevano/libmangal/logger"
	"github.com/philippgille/gokv"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"
)

var (
	_ Provider    = (*ProviderWithCache)(nil)
	_ LoginOption = (*CachedUserLoginOption)(nil)
)

// LoginOption is a per Provider login option that handles
// authorizing (internal to each provider) and getting the
// authorization token (for external caching).
type LoginOption interface {
	// String the name of the login option, for logging purposes.
	String() string

	// Token returns the authorization token (useful for caching).
	Token() *oauth2.Token
}

// TODO: only provide the token (this means get rid of the User caching)
// so that the providers check if the token is valid and refresh it
// as necessary
//
// CachedUserLoginOption is an implementation of LoginOption
// that provides cached data for a previously logged in User.
//
// This LoginOption is only meant to be created by the
// ProviderWithCache.LoginWithCachedUser method.
type CachedUserLoginOption struct {
	// User the cached User.
	User User

	token *oauth2.Token
}

// String the name of the login option, for logging purposes.
func (o *CachedUserLoginOption) String() string {
	return "Cached User Login"
}

// Token returns the authorization token (useful for caching).
func (o *CachedUserLoginOption) Token() *oauth2.Token {
	return o.token
}

// ProviderInfo is the passport of the metadata provider.
type ProviderInfo struct {
	// ID is the unique identifier of the provider.
	//
	// For a ProviderWithCache this is used as the cache DB name.
	ID IDCode `json:"id"`

	// Source is the source of the metadata.
	//
	// E.g. IDSourceAnilist, IDSourceMyAnimeList, etc.
	Source IDSource `json:"source"`

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

	// SetMangaProgress sets the reading progress for a given manga metadata id.
	SetMangaProgress(ctx context.Context, id, chapterNumber int) error

	// Authenticated returns true if the Provider is
	// currently authenticated (user logged in).
	Authenticated() bool

	// User returns the currently authenticated user.
	//
	// nil User means non-authenticated.
	User() User

	// Login authorizes an user with the given LoginOption.
	Login(ctx context.Context, loginOption LoginOption) error

	// Logout de-authorizes the currently authorized user.
	Logout() error
}

// ProviderWithCache is a Provider implementation with
// cache features, and extra search behavior.
//
// This is a wrapper on a normal Provider.
type ProviderWithCache struct {
	provider Provider
	store    store
	logger   *logger.Logger
}

// NewProviderWithCache constructs new Provider with cache given the Provider.
func NewProviderWithCache(options ProviderWithCacheOptions) (*ProviderWithCache, error) {
	if options.Provider == nil {
		return nil, Error("nil Provider passed to ProviderWithCache")
	}

	s := store{
		openStore: func(bucketName string) (gokv.Store, error) {
			return options.CacheStore(string(options.Provider.Info().ID), bucketName)
		},
	}

	// ensure the logger is non-nil
	l := options.Provider.Logger()
	if l == nil {
		l = logger.NewLogger()
		options.Provider.SetLogger(l)
	}

	p := &ProviderWithCache{
		provider: options.Provider,
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
func (p *ProviderWithCache) SetLogger(logger *logger.Logger) {
	p.logger = logger
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
	p.logger.Log("searching manga metadata with id %d on %q", id, p.Info().Name)
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

// TODO: implement cache for title (get single id by title if existent)?
//
// Search for metadata with the given query.
//
// Implementation should only handle the request and and marshaling.
func (p *ProviderWithCache) Search(ctx context.Context, query string) ([]Metadata, error) {
	p.logger.Log("searching manga metadata with query %q on %q", query, p.Info().Name)
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

// FindClosest metadata with the given title with its closest result.
func (p *ProviderWithCache) FindClosest(ctx context.Context, title string, tries, steps int) (Metadata, bool, error) {
	p.logger.Log("finding closest manga metadata with title %q on %q", title, p.Info().Name)

	id, found, err := p.store.getTitleID(title)
	if err != nil {
		return nil, false, Error(err.Error())
	}
	if found {
		meta, found, err := p.store.getMeta(id)
		if err != nil {
			return nil, false, Error(err.Error())
		}

		if found {
			return meta, true, nil
		}
	}

	meta, ok, err := p.findClosest(ctx, title, tries, steps)
	if err != nil {
		return nil, false, Error(err.Error())
	}
	if !ok {
		return nil, false, nil
	}

	id, err = strconv.Atoi(meta.ID().Raw)
	if err != nil {
		return nil, false, Error(err.Error())
	}
	err = p.store.setTitleID(title, id)
	if err != nil {
		return nil, false, Error(err.Error())
	}

	return meta, true, nil
}

func (p *ProviderWithCache) findClosest(ctx context.Context, title string, tries, step int) (Metadata, bool, error) {
	for i := 0; i < tries; i++ {
		p.logger.Log("finding closest try %d/%d", i+1, tries)

		metas, err := p.Search(ctx, title)
		if err != nil {
			return nil, false, err
		}

		if len(metas) > 0 {
			closest := metas[0]
			p.logger.Log("found closest: %q with id %d", closest.String(), closest.ID)
			return closest, true, nil
		}

		// try again with a different title
		// remove `step` characters from the end of the title
		// avoid removing the last character or going out of bounds
		var newLen int

		title = strings.TrimSpace(title)

		if len(title) > step {
			newLen = len(title) - step
		} else if len(title) > 1 {
			newLen = len(title) - 1
		} else {
			break
		}

		title = title[:newLen]
	}
	return nil, false, nil
}

// BindTitleWithID sets a given id to a title, so on each title search
// the same manga metadata with that id is obtained.
func (p *ProviderWithCache) BindTitleWithID(title string, id int) error {
	err := p.store.setTitleID(title, id)
	if err != nil {
		return Error(err.Error())
	}

	return nil
}

// SetMangaProgress sets the reading progress for a given manga metadata id.
//
// For ProviderWithCache this is only a wrapper around the actual provider's method.
func (p *ProviderWithCache) SetMangaProgress(ctx context.Context, id, chapterNumber int) error {
	return p.provider.SetMangaProgress(ctx, id, chapterNumber)
}

// Authenticated returns true if the Provider is
// currently authenticated (user logged in).
func (p *ProviderWithCache) Authenticated() bool {
	return p.provider.Authenticated()
}

// User returns the currently authenticated user.
//
// nil User means non-authenticated.
func (p *ProviderWithCache) User() User {
	return p.provider.User()
}

// Login authorizes an user with the given credentials.
func (p *ProviderWithCache) Login(ctx context.Context, loginOption LoginOption) error {
	p.logger.Log("authenticating Provider %q with login option %q", p.Info().Name, loginOption)
	err := p.provider.Login(ctx, loginOption)
	if err != nil {
		return AuthError(err.Error())
	}

	// Get the just authorized data
	user := p.User()
	if user == nil {
		return AuthError("got nil User when retrieving from Provider " + p.Info().Name)
	}
	token := loginOption.Token()
	if token == nil {
		return AuthError("got nil Token when retrieving from Provider " + p.Info().Name)
	}

	// Store the user and auth data to cache
	if err := p.store.setUser(user.Name(), user); err != nil {
		return AuthError(err.Error())
	}
	if err := p.store.setToken(user.Name(), *token); err != nil {
		return AuthError(err.Error())
	}

	return nil
}

// Logout de-authorizes the currently authorized user.
func (p *ProviderWithCache) Logout() error {
	return p.provider.Logout()
}

// TODO: handle re-caching the token if it is refreshed (due to it expiring)
// TODO: change method signature from returning (bool, error) to returning error only
//
// LoginWithCachedUser will try to get the cached authentication data for the given
// username. If the data exists, the provider will be authenticated with this user.
func (p *ProviderWithCache) LoginWithCachedUser(ctx context.Context, username string) (bool, error) {
	p.logger.Log("authenticating Provider %q via cached user %q", p.Info().Name, username)
	// Get stored authData
	token, found, err := p.store.getToken(username)
	if err != nil {
		return false, AuthError(err.Error())
	}
	if !found {
		return false, nil
	}

	// Get stored user (if there is a token, there should be an user)
	user, found, err := p.store.getUser(username)
	if err != nil {
		return false, AuthError(err.Error())
	}
	// If no user found, delete the found access token
	if !found {
		p.logger.Log("cached access token for user %q found but there is no cached user data, need to re-authenticate", username)
		err := p.store.deleteToken(username)
		if err != nil {
			return false, AuthError(err.Error())
		}
		return false, nil
	}

	loginOption := &CachedUserLoginOption{
		User:  user,
		token: &token,
	}

	err = p.provider.Login(ctx, loginOption)
	if err != nil {
		return false, err
	}

	return true, nil
}

// DeleteCachedUser will delete the specified user cached auth data.
func (p *ProviderWithCache) DeleteCachedUser(username string) error {
	p.logger.Log("deleting cached authentication user data for %q for Provider %q", username, p.Info().Name)
	err := p.store.deleteUser(username)
	if err != nil {
		return err
	}
	err = p.store.deleteToken(username)
	if err != nil {
		return err
	}
	return nil
}
