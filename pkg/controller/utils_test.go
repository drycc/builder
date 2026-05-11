package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	tokenpkg "github.com/drycc/builder/pkg/controller/token"
	drycc "github.com/drycc/controller-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

// installTokenForTest replaces the package-level token Manager with one
// backed by miniredis + a stub passport server, isolating tests from each
// other and from any real DRYCC_VALKEY_URL in the environment.
func installTokenForTest(t *testing.T, passport http.HandlerFunc) (*miniredis.Miniredis, *httptest.Server) {
	t.Helper()
	mr := miniredis.RunT(t)
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(client.Close)

	ts := httptest.NewServer(passport)
	t.Cleanup(ts.Close)
	t.Setenv("DRYCC_PASSPORT_URL", ts.URL)
	t.Setenv("DRYCC_PASSPORT_KEY", "k")
	t.Setenv("DRYCC_PASSPORT_SECRET", "s")
	t.Setenv("DRYCC_VALKEY_URL", "redis://"+mr.Addr())

	tokenpkg.ResetForTest(t, client)
	return mr, ts
}

func TestNew_PullsTokenFromValkeyAndSetsBearer(t *testing.T) {
	installTokenForTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"abc","token_type":"Bearer","expires_in":2592000}`))
	})

	cli, err := New(context.Background(), "http://controller.example/")
	require.NoError(t, err)
	assert.Equal(t, "drycc-builder", cli.UserAgent)
	assert.Empty(t, cli.Token, "Token must be empty; authTransport owns Authorization")
}

func TestNew_RetriesOnceOn401(t *testing.T) {
	tokens := []string{"first", "second"}
	var passportCalls int32
	passport := func(w http.ResponseWriter, _ *http.Request) {
		idx := int(atomic.AddInt32(&passportCalls, 1) - 1)
		if idx >= len(tokens) {
			idx = len(tokens) - 1
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": tokens[idx],
			"token_type":   "Bearer",
			"expires_in":   2592000,
		})
	}
	installTokenForTest(t, passport)

	var controllerCalls int32
	var seenAuth []string
	controllerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = append(seenAuth, r.Header.Get("Authorization"))
		if atomic.AddInt32(&controllerCalls, 1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("DRYCC_API_VERSION", drycc.APIVersion)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	}))
	defer controllerSrv.Close()

	cli, err := New(context.Background(), controllerSrv.URL)
	require.NoError(t, err)

	resp, err := cli.Request(http.MethodGet, "/v2/ping", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), atomic.LoadInt32(&controllerCalls), "expected exactly one retry")
	require.Len(t, seenAuth, 2)
	assert.Equal(t, "Bearer first", seenAuth[0])
	assert.Equal(t, "Bearer second", seenAuth[1])
	assert.Equal(t, int32(2), atomic.LoadInt32(&passportCalls), "passport hit on cold start + 401 refresh")
}

func TestNew_PropagatesUnauthorizedAfterRetry(t *testing.T) {
	installTokenForTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"always-bad","token_type":"Bearer","expires_in":2592000}`))
	})

	controllerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer controllerSrv.Close()

	cli, err := New(context.Background(), controllerSrv.URL)
	require.NoError(t, err)

	_, err = cli.Request(http.MethodGet, "/v2/ping", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, drycc.ErrUnauthorized)
}

func TestCheckAPICompat(t *testing.T) {
	client := &drycc.Client{ControllerAPIVersion: drycc.APIVersion}
	if apiErr := CheckAPICompat(client, drycc.ErrAPIMismatch); apiErr != nil {
		t.Errorf("api errors are non-fatal and should return nil, got '%v'", apiErr)
	}
	if apiErr := CheckAPICompat(client, errors.New("random error")); apiErr == nil {
		t.Error("expected error to be returned, got nil")
	}
}

func TestNew_PostBodyIsReplayedOn401(t *testing.T) {
	tokens := []string{"first", "second"}
	var passportCalls int32
	installTokenForTest(t, func(w http.ResponseWriter, _ *http.Request) {
		idx := int(atomic.AddInt32(&passportCalls, 1) - 1)
		if idx >= len(tokens) {
			idx = len(tokens) - 1
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": tokens[idx],
			"token_type":   "Bearer",
			"expires_in":   2592000,
		})
	})

	const wantBody = `{"app":"hello","build":"sha256:abc"}`
	var seenBodies []string
	controllerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seenBodies = append(seenBodies, string(body))
		if len(seenBodies) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("DRYCC_API_VERSION", drycc.APIVersion)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer controllerSrv.Close()

	cli, err := New(context.Background(), controllerSrv.URL)
	require.NoError(t, err)

	resp, err := cli.Request(http.MethodPost, "/v2/builds/", []byte(wantBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	require.Len(t, seenBodies, 2, "expected initial request + one retry")
	assert.Equal(t, wantBody, seenBodies[0], "first attempt body must arrive intact")
	assert.Equal(t, wantBody, seenBodies[1], "replayed body must be byte-identical")
}

func TestNew_NetworkErrorDoesNotTriggerRetry(t *testing.T) {
	installTokenForTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"t","token_type":"Bearer","expires_in":2592000}`))
	})

	cli, err := New(context.Background(), "http://127.0.0.1:1")
	require.NoError(t, err)

	_, err = cli.Request(http.MethodGet, "/v2/ping", nil)
	require.Error(t, err)
	assert.NotErrorIs(t, err, drycc.ErrUnauthorized)
}

func TestNew_SuccessfulRequestReusesToken(t *testing.T) {
	var passportCalls int32
	installTokenForTest(t, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&passportCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"reuse-me","token_type":"Bearer","expires_in":2592000}`))
	})

	var authHeaders []string
	controllerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		w.Header().Set("DRYCC_API_VERSION", drycc.APIVersion)
		w.WriteHeader(http.StatusOK)
	}))
	defer controllerSrv.Close()

	cli, err := New(context.Background(), controllerSrv.URL)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		resp, err := cli.Request(http.MethodGet, "/v2/ping", nil)
		require.NoError(t, err)
		_ = resp.Body.Close()
	}

	assert.Equal(t, int32(1), atomic.LoadInt32(&passportCalls), "passport must only be hit on the cold start")
	require.Len(t, authHeaders, 3)
	for _, h := range authHeaders {
		assert.Equal(t, "Bearer reuse-me", h)
	}
}
