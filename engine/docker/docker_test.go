package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRoundTripper struct {
	requests []*http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)

	path := stripAPIVersion(req.URL)

	// Handle endpoints used by Engine
	switch {
	case req.Method == http.MethodPost && path == "/containers/create":
		payload := struct {
			ID       string   `json:"Id"`
			Warnings []string `json:"Warnings"`
		}{ID: "abc123"}
		body, _ := json.Marshal(payload)
		return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil

	case req.Method == http.MethodPost && strings.HasSuffix(path, "/containers/abc123/start"):
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil

	case req.Method == http.MethodDelete && strings.HasPrefix(path, "/containers/abc123"):
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil

	case req.Method == http.MethodGet && path == "/images/json":
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("[]")), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	}

	return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func stripAPIVersion(u *url.URL) string {
	// Remove "/vX[.Y]" prefix if present
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) > 0 && strings.HasPrefix(parts[0], "v") {
		return "/" + strings.Join(parts[1:], "/")
	}
	return u.Path
}

func newMockDockerClient(t *testing.T) *client.Client {
	t.Helper()
	rt := &mockRoundTripper{}
	hc := &http.Client{Transport: rt}
	cli, err := client.NewClientWithOpts(
		client.WithHost("http://fake"),
		client.WithHTTPClient(hc),
		client.WithVersion("1.43"),
	)
	if err != nil {
		t.Fatalf("failed to create mock docker client: %v", err)
	}
	return cli
}

func TestCreate_ReturnsContainerID(t *testing.T) {
	cli := newMockDockerClient(t)
	eng := &Engine{client: cli, cfg: &Config{PullPolicy: ""}}

	id, err := eng.Create(context.Background(), "busybox:latest")
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, "abc123", *id)
}

func TestRemove_RemovesContainer(t *testing.T) {
	cli := newMockDockerClient(t)
	eng := &Engine{client: cli, cfg: &Config{}}

	err := eng.Remove(context.Background(), "abc123")
	require.NoError(t, err)
}

func TestCreate_WithBasicConfig(t *testing.T) {
	cli := newMockDockerClient(t)
	eng := &Engine{client: cli, cfg: &Config{DefaultContainer: ContainerConfig{Labels: map[string]string{"app": "test"}, Cmd: []string{"echo", "hi"}}}}

	containerCfg, hostCfg, netCfg, name, err := eng.buildContainerConfigs("busybox:latest")
	require.NoError(t, err)
	assert.Equal(t, "busybox:latest", containerCfg.Image)
	_ = hostCfg
	_ = netCfg
	_ = name

	// Ensure create works end-to-end with this config
	id, err := eng.Create(context.Background(), "busybox:latest")
	require.NoError(t, err)
	require.NotNil(t, id)
	assert.Equal(t, "abc123", *id)
}
