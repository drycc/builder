package token

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

func newTestManager(t *testing.T, passportHandler http.HandlerFunc) (*Manager, *miniredis.Miniredis, *httptest.Server) {
	t.Helper()

	mr := miniredis.RunT(t)

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(client.Close)

	ts := httptest.NewServer(passportHandler)
	t.Cleanup(ts.Close)

	t.Setenv("DRYCC_PASSPORT_URL", ts.URL)
	t.Setenv("DRYCC_PASSPORT_KEY", "test-key")
	t.Setenv("DRYCC_PASSPORT_SECRET", "test-secret")
	t.Setenv("DRYCC_PASSPORT_SCOPES", "controller:hook")

	mgr, err := NewManager(client)
	require.NoError(t, err)
	return mgr, mr, ts
}

func passportJSON(t *testing.T, accessToken string, expiresIn int64) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
			"scope":        "passport:message",
		})
	}
}

func TestGet_FastPathReturnsCachedToken(t *testing.T) {
	mgr, mr, _ := newTestManager(t, passportJSON(t, "should-not-be-fetched", 2592000))

	// Pre-populate Valkey with a still-valid token.
	p := payload{AccessToken: "cached-token", ExpiresAt: time.Now().Add(24 * time.Hour).Unix()}
	raw, _ := json.Marshal(p)
	require.NoError(t, mr.Set(TokenKey, string(raw)))

	got, err := mgr.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "cached-token", got)
}

func TestGet_ColdStartFetchesFromPassport(t *testing.T) {
	var calls int32
	handler := func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fresh","token_type":"Bearer","expires_in":2592000,"scope":"passport:message"}`))
	}
	mgr, mr, _ := newTestManager(t, handler)

	got, err := mgr.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "fresh", got)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))

	stored, err := mr.Get(TokenKey)
	require.NoError(t, err)
	var p payload
	require.NoError(t, json.Unmarshal([]byte(stored), &p))
	assert.Equal(t, "fresh", p.AccessToken)
	assert.InDelta(t, time.Now().Add(30*24*time.Hour).Unix(), p.ExpiresAt, 5)

	ttl := mr.TTL(TokenKey)
	expected := 30*24*time.Hour + ValkeyTTLBuffer
	assert.InDelta(t, expected.Seconds(), ttl.Seconds(), 5)
}

func TestGet_ConcurrentCallsHitPassportOnce(t *testing.T) {
	var calls int32
	handler := func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		// Slow handler to widen the race window.
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"only-one","token_type":"Bearer","expires_in":2592000}`))
	}
	mgr, _, _ := newTestManager(t, handler)

	var wg sync.WaitGroup
	const n = 20
	results := make([]string, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = mgr.Get(context.Background())
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		require.NoErrorf(t, errs[i], "goroutine %d", i)
		assert.Equal(t, "only-one", results[i])
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "passport should be called exactly once")
}

func TestRefresh_SkipsWhenPlentyOfLifetimeRemains(t *testing.T) {
	var calls int32
	handler := func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"new","expires_in":2592000}`))
	}
	mgr, mr, _ := newTestManager(t, handler)

	// Token with 20 days remaining (> RefreshThreshold of 7d).
	p := payload{AccessToken: "still-good", ExpiresAt: time.Now().Add(20 * 24 * time.Hour).Unix()}
	raw, _ := json.Marshal(p)
	require.NoError(t, mr.Set(TokenKey, string(raw)))

	require.NoError(t, mgr.Refresh(context.Background(), false))
	assert.Equal(t, int32(0), atomic.LoadInt32(&calls))

	got, _ := mr.Get(TokenKey)
	assert.Equal(t, string(raw), got, "token must be untouched")
}

func TestRefresh_RefreshesWhenCloseToExpiry(t *testing.T) {
	mgr, mr, _ := newTestManager(t, passportJSON(t, "renewed", 2592000))

	// Less than RefreshThreshold remaining.
	p := payload{AccessToken: "old", ExpiresAt: time.Now().Add(2 * 24 * time.Hour).Unix()}
	raw, _ := json.Marshal(p)
	require.NoError(t, mr.Set(TokenKey, string(raw)))

	require.NoError(t, mgr.Refresh(context.Background(), false))

	stored, _ := mr.Get(TokenKey)
	var got payload
	require.NoError(t, json.Unmarshal([]byte(stored), &got))
	assert.Equal(t, "renewed", got.AccessToken)
}

func TestRefresh_ForceAlwaysRefreshes(t *testing.T) {
	var calls int32
	handler := func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"forced","expires_in":2592000}`))
	}
	mgr, mr, _ := newTestManager(t, handler)

	p := payload{AccessToken: "still-fresh", ExpiresAt: time.Now().Add(29 * 24 * time.Hour).Unix()}
	raw, _ := json.Marshal(p)
	require.NoError(t, mr.Set(TokenKey, string(raw)))

	require.NoError(t, mgr.Refresh(context.Background(), true))
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))

	stored, _ := mr.Get(TokenKey)
	var got payload
	require.NoError(t, json.Unmarshal([]byte(stored), &got))
	assert.Equal(t, "forced", got.AccessToken)
}

func TestRefresh_NoCachedTokenFetchesAnyway(t *testing.T) {
	var calls int32
	handler := func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte(`{"access_token":"bootstrap","expires_in":2592000}`))
	}
	mgr, _, _ := newTestManager(t, handler)

	require.NoError(t, mgr.Refresh(context.Background(), false))
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestRefresh_MissingExpiresInUsesDefault(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"no-expiry","token_type":"Bearer"}`))
	}
	mgr, mr, _ := newTestManager(t, handler)

	require.NoError(t, mgr.Refresh(context.Background(), true))

	stored, _ := mr.Get(TokenKey)
	var p payload
	require.NoError(t, json.Unmarshal([]byte(stored), &p))
	assert.InDelta(t, time.Now().Add(DefaultExpiresIn).Unix(), p.ExpiresAt, 5)
}

func TestInvalidate_ClearsToken(t *testing.T) {
	mgr, mr, _ := newTestManager(t, passportJSON(t, "irrelevant", 2592000))

	p := payload{AccessToken: "doomed", ExpiresAt: time.Now().Add(24 * time.Hour).Unix()}
	raw, _ := json.Marshal(p)
	require.NoError(t, mr.Set(TokenKey, string(raw)))

	require.NoError(t, mgr.Invalidate(context.Background()))
	assert.False(t, mr.Exists(TokenKey))
}

func TestGet_ExpiredTokenTriggersRefresh(t *testing.T) {
	mgr, mr, _ := newTestManager(t, passportJSON(t, "after-expiry", 2592000))

	// Already expired.
	p := payload{AccessToken: "stale", ExpiresAt: time.Now().Add(-time.Hour).Unix()}
	raw, _ := json.Marshal(p)
	require.NoError(t, mr.Set(TokenKey, string(raw)))

	got, err := mgr.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "after-expiry", got)
}

func TestGet_PassportFailurePropagates(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}
	mgr, _, _ := newTestManager(t, handler)

	_, err := mgr.Get(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestNewManager_MissingEnvFails(t *testing.T) {
	t.Setenv("DRYCC_PASSPORT_URL", "")
	t.Setenv("DRYCC_PASSPORT_KEY", "")
	t.Setenv("DRYCC_PASSPORT_SECRET", "")

	mr := miniredis.RunT(t)
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(client.Close)

	_, err = NewManager(client)
	require.Error(t, err)
}

func TestGet_CorruptedJSONIsTreatedAsMiss(t *testing.T) {
	mgr, mr, _ := newTestManager(t, passportJSON(t, "recovered", 2592000))

	require.NoError(t, mr.Set(TokenKey, "{not valid json"))

	got, err := mgr.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "recovered", got, "corrupted entry must trigger refresh, not propagate")

	stored, _ := mr.Get(TokenKey)
	var p payload
	require.NoError(t, json.Unmarshal([]byte(stored), &p), "corrupted blob must be overwritten with valid JSON")
}

func TestRefresh_CorruptedJSONForcesRefresh(t *testing.T) {
	var calls int32
	handler := func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte(`{"access_token":"after-corrupt","expires_in":2592000}`))
	}
	mgr, mr, _ := newTestManager(t, handler)

	require.NoError(t, mr.Set(TokenKey, "garbage"))

	require.NoError(t, mgr.Refresh(context.Background(), false))
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestAcquireLock_TimesOutWhenHeldByAnother(t *testing.T) {
	prevTimeout := lockBlockingTimeout
	prevPoll := lockPollInterval
	lockBlockingTimeout = 200 * time.Millisecond
	lockPollInterval = 30 * time.Millisecond
	t.Cleanup(func() {
		lockBlockingTimeout = prevTimeout
		lockPollInterval = prevPoll
	})

	mgr, mr, _ := newTestManager(t, passportJSON(t, "never-fetched", 2592000))

	require.NoError(t, mr.Set(InitLockKey, "someone-else"))
	mr.SetTTL(InitLockKey, InitLockTTL)

	start := time.Now()
	_, err := mgr.Get(context.Background())
	elapsed := time.Since(start)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout waiting for token refresh lock")
	assert.GreaterOrEqual(t, elapsed, 200*time.Millisecond)
	assert.Less(t, elapsed, 2*time.Second, "should not wait the full production timeout")
}

func TestReleaseLock_DoesNotDeleteWhenOwnerMismatches(t *testing.T) {
	mgr, mr, _ := newTestManager(t, passportJSON(t, "x", 2592000))

	require.NoError(t, mr.Set(InitLockKey, "another-owner"))
	mr.SetTTL(InitLockKey, InitLockTTL)

	mgr.releaseLock(context.Background(), "our-owner")

	val, err := mr.Get(InitLockKey)
	require.NoError(t, err)
	assert.Equal(t, "another-owner", val, "Lua owner-check must protect foreign locks")
}

func TestReleaseLock_DeletesWhenOwnerMatches(t *testing.T) {
	mgr, mr, _ := newTestManager(t, passportJSON(t, "x", 2592000))

	require.NoError(t, mr.Set(InitLockKey, "us"))
	mr.SetTTL(InitLockKey, InitLockTTL)

	mgr.releaseLock(context.Background(), "us")

	assert.False(t, mr.Exists(InitLockKey))
}

func TestGet_ContextCancellationPropagates(t *testing.T) {
	prevTimeout := lockBlockingTimeout
	prevPoll := lockPollInterval
	lockBlockingTimeout = 5 * time.Second
	lockPollInterval = 50 * time.Millisecond
	t.Cleanup(func() {
		lockBlockingTimeout = prevTimeout
		lockPollInterval = prevPoll
	})

	mgr, mr, _ := newTestManager(t, passportJSON(t, "x", 2592000))

	require.NoError(t, mr.Set(InitLockKey, "blocker"))
	mr.SetTTL(InitLockKey, InitLockTTL)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := mgr.Get(ctx)
	elapsed := time.Since(start)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, 1*time.Second, "ctx cancel must abort the lock loop promptly")
}

func TestNewClientFromURL(t *testing.T) {
	mr := miniredis.RunT(t)
	c, err := NewClientFromURL("redis://" + mr.Addr())
	require.NoError(t, err)
	require.NoError(t, c.Do(context.Background(), c.B().Ping().Build()).Error())
	c.Close()
}

func TestNewClientFromEnv_MissingURL(t *testing.T) {
	t.Setenv("DRYCC_VALKEY_URL", "")
	_, err := NewClientFromEnv()
	require.Error(t, err)
}

func TestRequestToken_SendsControllerHookScope(t *testing.T) {
	var captured url.Values
	handler := func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ok","token_type":"Bearer","expires_in":2592000}`))
	}
	mgr, _, _ := newTestManager(t, handler)

	_, err := mgr.Get(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "client_credentials", captured.Get("grant_type"))
	assert.Equal(t, "test-key", captured.Get("client_id"))
	assert.Equal(t, "test-secret", captured.Get("client_secret"))
	assert.Equal(t, "controller:hook", captured.Get("scope"))
}
