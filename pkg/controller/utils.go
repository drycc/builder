// Package controller provides utilities for interacting with the Drycc controller.
package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/drycc/builder/pkg/controller/token"
	drycc "github.com/drycc/controller-sdk-go"
	"github.com/drycc/pkg/log"
)

// New creates a new SDK client configured as the builder. The OAuth m2m
// access token is sourced from Valkey via the token package; the HTTP
// transport transparently re-fetches on 401 (self-heal).
func New(ctx context.Context, controllerURL string) (*drycc.Client, error) {
	client, err := drycc.New(true, controllerURL, "")
	if err != nil {
		return client, err
	}
	client.UserAgent = "drycc-builder"

	tok, err := token.Get(ctx)
	if err != nil {
		return client, err
	}

	base := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: !client.VerifySSL},
		DisableKeepAlives: true,
		Proxy:             http.ProxyFromEnvironment,
	}
	at := &authTransport{base: base}
	at.token.Store(tok)
	client.HTTPClient = &http.Client{Transport: at}

	// Clear Token so the SDK does not append its own Authorization header;
	// authTransport is the sole owner of Authorization.
	client.Token = ""
	return client, nil
}

// CheckAPICompat checks for API compatibility errors and warns about them.
func CheckAPICompat(c *drycc.Client, err error) error {
	if errors.Is(err, drycc.ErrAPIMismatch) {
		log.Info("WARNING: SDK and Controller API versions do not match. SDK: %s Controller: %s",
			drycc.APIVersion, c.ControllerAPIVersion)
		return nil
	}
	return err
}

// authTransport injects the cached bearer token on every request and, on
// 401, invalidates the cache and replays the request exactly once with a
// freshly fetched token. This is the runtime self-heal path: if the CronJob
// is late or the cached token was rotated out-of-band, in-flight requests
// recover without the caller noticing.
type authTransport struct {
	base  http.RoundTripper
	token atomic.Value
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := copyBody(req)
	if err != nil {
		return nil, err
	}
	t.setAuth(req)
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	_ = resp.Body.Close()
	if err := token.Invalidate(req.Context()); err != nil {
		log.Info("token: failed to invalidate after 401: %s", err)
	}
	fresh, err := token.Get(req.Context())
	if err != nil {
		return nil, err
	}
	t.token.Store(fresh)
	retry := req.Clone(req.Context())
	if body != nil {
		retry.Body = io.NopCloser(bytes.NewReader(body))
		retry.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}
	t.setAuth(retry)
	return t.base.RoundTrip(retry)
}

func (t *authTransport) setAuth(req *http.Request) {
	if tok, ok := t.token.Load().(string); ok && tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

// copyBody buffers a request body so the same payload can be replayed after
// a 401. Returns nil for bodyless requests.
func copyBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	buf, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(buf))
	return buf, nil
}
