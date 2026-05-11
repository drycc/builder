// Package token manages OAuth m2m access tokens for the Drycc builder.
//
// It is a Go port of grafana/rootfs/usr/share/grafana/oauth2/token.py, the
// reference implementation used by drycc/grafana. The token is persisted in
// Valkey so that all builder replicas share a single cached credential, and a
// Kubernetes CronJob refreshes it well before expiry. The runtime (sshd,
// healthsrv, gitreceive) only reads from Valkey on the fast path; a
// distributed lock guards the rare cache-miss / cold-start path so that the
// passport service is hit at most once.
package token

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

// Tunables mirror the grafana token.py constants. Exported so tests can
// override them without poking package internals.
const (
	// TokenKey is the Valkey key under which the JSON-encoded token lives.
	TokenKey = "builder:oauth2:token"
	// InitLockKey guards concurrent refreshes during cold start.
	InitLockKey = "builder:oauth2:init_lock"
	// InitLockTTL bounds how long the refresh critical section may run.
	InitLockTTL = 30 * time.Second
	// RefreshThreshold tells the CronJob to refresh when less than this
	// much lifetime remains.
	RefreshThreshold = 7 * 24 * time.Hour
	// ReadBuffer is the safety margin used by Get(): a token whose expiry
	// is within this window is treated as already expired.
	ReadBuffer = 60 * time.Second
	// DefaultExpiresIn is the fallback when passport omits expires_in.
	DefaultExpiresIn = 30 * 24 * time.Hour
	// ValkeyTTLBuffer extends the Valkey TTL past the OAuth expiry so the
	// CronJob has a chance to refresh before the key disappears.
	ValkeyTTLBuffer = 7 * 24 * time.Hour
)

// Lock-loop timings. Vars (not consts) so tests can shorten them; not part
// of the public API contract.
var (
	lockPollInterval    = 200 * time.Millisecond
	lockBlockingTimeout = 30 * time.Second
)

// payload is the JSON shape stored in Valkey. Identical to grafana token.py.
type payload struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"` // unix seconds
}

// tokenResponse models the passport /oauth/token/ response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// Manager owns the Valkey client and the configuration needed to refresh
// tokens. It is safe for concurrent use.
type Manager struct {
	client         valkey.Client
	passportURL    string
	passportKey    string
	passportSecret string
	passportScopes string
	httpClient     *http.Client
	now            func() time.Time
}

// Option customises a Manager. Used mostly by tests.
type Option func(*Manager)

// WithHTTPClient overrides the http.Client used to talk to passport.
func WithHTTPClient(c *http.Client) Option {
	return func(m *Manager) { m.httpClient = c }
}

// WithClock overrides the time source. Tests use this to make TTL maths
// deterministic.
func WithClock(now func() time.Time) Option {
	return func(m *Manager) { m.now = now }
}

// NewManager builds a Manager from environment variables and an explicit
// valkey-go client. The caller owns the client and must Close() it.
//
// Required env vars: DRYCC_PASSPORT_URL, DRYCC_PASSPORT_KEY,
// DRYCC_PASSPORT_SECRET. They match the existing builder convention.
func NewManager(client valkey.Client, opts ...Option) (*Manager, error) {
	passportURL := os.Getenv("DRYCC_PASSPORT_URL")
	passportKey := os.Getenv("DRYCC_PASSPORT_KEY")
	passportSecret := os.Getenv("DRYCC_PASSPORT_SECRET")
	passportScopes := os.Getenv("DRYCC_PASSPORT_SCOPES")
	if passportURL == "" || passportKey == "" || passportSecret == "" {
		return nil, errors.New("passport credentials not configured")
	}
	m := &Manager{
		client:         client,
		passportURL:    passportURL,
		passportKey:    passportKey,
		passportSecret: passportSecret,
		passportScopes: passportScopes,
		httpClient:     http.DefaultClient,
		now:            time.Now,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

// NewClientFromEnv constructs a valkey-go client from DRYCC_VALKEY_URL. The
// URL follows the Drycc convention, e.g.
//
//	redis://:password@drycc-valkey:16379/2
//
// Both redis:// and valkey:// schemes are accepted.
func NewClientFromEnv() (valkey.Client, error) {
	raw := os.Getenv("DRYCC_VALKEY_URL")
	if raw == "" {
		return nil, errors.New("DRYCC_VALKEY_URL not set")
	}
	return NewClientFromURL(raw)
}

// NewClientFromURL parses a redis-style URL into a valkey-go client.
// Client-side caching is disabled because the runtime keeps no per-process
// cache of its own and not every Valkey/Redis flavour ships RESP3 tracking.
func NewClientFromURL(raw string) (valkey.Client, error) {
	opt, err := valkey.ParseURL(raw)
	if err != nil {
		u, perr := url.Parse(raw)
		if perr != nil {
			return nil, fmt.Errorf("invalid DRYCC_VALKEY_URL: %w", err)
		}
		opt = valkey.ClientOption{InitAddress: []string{u.Host}}
		if u.User != nil {
			if pw, ok := u.User.Password(); ok {
				opt.Password = pw
			}
			opt.Username = u.User.Username()
		}
	}
	opt.DisableCache = true
	return valkey.NewClient(opt)
}

// Get returns a currently-valid access token, performing a synchronous
// refresh through the distributed lock if Valkey has nothing usable. This is
// the runtime fast path used by sshd/healthsrv/gitreceive. Mirrors
// grafana token.py::get_token().
func (m *Manager) Get(ctx context.Context) (string, error) {
	// 1. Fast path: try to read a valid token directly.
	if p, err := m.readValid(ctx, ReadBuffer); err == nil && p != nil {
		return p.AccessToken, nil
	}

	// 2. Acquire the distributed lock with bounded wait.
	owner, err := m.acquireLock(ctx)
	if err != nil {
		return "", err
	}
	defer m.releaseLock(context.Background(), owner)

	// 3. Double-check: another caller may have refreshed while we waited.
	if p, err := m.readValid(ctx, ReadBuffer); err == nil && p != nil {
		return p.AccessToken, nil
	}

	// 4. Cold path: fetch from passport and persist.
	p, err := m.fetchAndSave(ctx)
	if err != nil {
		return "", err
	}
	return p.AccessToken, nil
}

// Refresh is the CronJob entry point. When force is true the token is
// refreshed unconditionally; otherwise the token is left alone unless less
// than RefreshThreshold of lifetime remains. Mirrors token.py::async_main().
func (m *Manager) Refresh(ctx context.Context, force bool) error {
	if !force {
		p, err := m.readValid(ctx, 0)
		if err != nil {
			return err
		}
		if p != nil {
			remaining := time.Until(time.Unix(p.ExpiresAt, 0))
			if remaining > RefreshThreshold {
				return nil
			}
		}
	}
	_, err := m.fetchAndSave(ctx)
	return err
}

// Invalidate deletes the cached token so the next Get() forces a refresh.
// Used by the 401 self-heal path.
func (m *Manager) Invalidate(ctx context.Context) error {
	return m.client.Do(ctx, m.client.B().Del().Key(TokenKey).Build()).Error()
}

// ---- internals -----------------------------------------------------------

func (m *Manager) readValid(ctx context.Context, buffer time.Duration) (*payload, error) {
	resp := m.client.Do(ctx, m.client.B().Get().Key(TokenKey).Build())
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, nil
		}
		return nil, err
	}
	raw, err := resp.ToString()
	if err != nil {
		return nil, err
	}
	var p payload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		// Corrupted entry: treat as miss; CronJob/Get will rewrite it.
		return nil, nil
	}
	cutoff := m.now().Add(buffer).Unix()
	if p.ExpiresAt <= cutoff {
		return nil, nil
	}
	return &p, nil
}

func (m *Manager) fetchAndSave(ctx context.Context) (*payload, error) {
	tr, err := m.requestToken(ctx)
	if err != nil {
		return nil, err
	}
	expiresIn := time.Duration(tr.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = DefaultExpiresIn
	}
	now := m.now()
	p := payload{
		AccessToken: tr.AccessToken,
		ExpiresAt:   now.Add(expiresIn).Unix(),
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	ttl := expiresIn + ValkeyTTLBuffer
	err = m.client.Do(ctx,
		m.client.B().Set().Key(TokenKey).Value(string(raw)).
			ExSeconds(int64(ttl.Seconds())).Build(),
	).Error()
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (m *Manager) requestToken(ctx context.Context) (*tokenResponse, error) {
	endpoint := strings.TrimRight(m.passportURL, "/") + "/oauth/token/"
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", m.passportKey)
	form.Set("client_secret", m.passportSecret)
	form.Set("scope", m.passportScopes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("passport returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, errors.New("passport returned empty access_token")
	}
	return &tr, nil
}

// acquireLock takes InitLockKey with SET NX EX, retrying up to
// lockBlockingTimeout. The returned string is the owner token; pass it to
// releaseLock so we never delete a lock that someone else now holds.
func (m *Manager) acquireLock(ctx context.Context) (string, error) {
	owner := uuid.NewString()
	deadline := time.Now().Add(lockBlockingTimeout)
	for {
		err := m.client.Do(ctx,
			m.client.B().Set().Key(InitLockKey).Value(owner).
				Nx().ExSeconds(int64(InitLockTTL.Seconds())).Build(),
		).Error()
		if err == nil {
			return owner, nil
		}
		// valkey-go signals "NX rejected" as a Nil reply, not an error
		// string. Anything else is fatal.
		if !valkey.IsValkeyNil(err) {
			return "", fmt.Errorf("acquire init lock: %w", err)
		}
		if time.Now().After(deadline) {
			return "", errors.New("timeout waiting for token refresh lock")
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(lockPollInterval):
		}
	}
}

// releaseLock deletes InitLockKey only if we still own it. Uses a Lua check
// to avoid a TOCTOU where our lock expired and someone else grabbed it.
func (m *Manager) releaseLock(ctx context.Context, owner string) {
	const script = `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) else return 0 end`
	_ = m.client.Do(ctx,
		m.client.B().Eval().Script(script).Numkeys(1).
			Key(InitLockKey).Arg(owner).Build(),
	).Error()
}

// ---- package-level singleton --------------------------------------------
//
// The runtime call sites (sshd/healthsrv/gitreceive) want a tiny API:
// token.Get(ctx) / token.Invalidate(ctx). We lazily construct a Manager from
// the standard env vars on first use.

var (
	defaultOnce sync.Once
	defaultMgr  *Manager
	defaultErr  error
)

func getDefault() (*Manager, error) {
	defaultOnce.Do(func() {
		client, err := NewClientFromEnv()
		if err != nil {
			defaultErr = err
			return
		}
		mgr, err := NewManager(client)
		if err != nil {
			client.Close()
			defaultErr = err
			return
		}
		defaultMgr = mgr
	})
	return defaultMgr, defaultErr
}

// Get is a convenience wrapper around the package-level Manager.
func Get(ctx context.Context) (string, error) {
	mgr, err := getDefault()
	if err != nil {
		return "", err
	}
	return mgr.Get(ctx)
}

// Refresh is a convenience wrapper around the package-level Manager.
func Refresh(ctx context.Context, force bool) error {
	mgr, err := getDefault()
	if err != nil {
		return err
	}
	return mgr.Refresh(ctx, force)
}

// Invalidate is a convenience wrapper around the package-level Manager.
func Invalidate(ctx context.Context) error {
	mgr, err := getDefault()
	if err != nil {
		return err
	}
	return mgr.Invalidate(ctx)
}

// ResetForTest replaces the package-level Manager with one bound to the
// supplied Valkey client. It re-reads passport env vars so tests can swap
// them via t.Setenv. Test-only; not part of the public API contract.
func ResetForTest(t interface{ Helper() }, client valkey.Client) {
	t.Helper()
	defaultOnce = sync.Once{}
	defaultMgr = nil
	defaultErr = nil
	mgr, err := NewManager(client)
	defaultMgr, defaultErr = mgr, err
	defaultOnce.Do(func() {})
}
