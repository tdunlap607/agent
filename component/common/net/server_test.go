package net

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/grafana/agent/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestTargetServer(t *testing.T) {
	// dependencies
	reg := prometheus.NewRegistry()
	ts, err := NewTargetServer(util.TestLogger(t), "test_namespace", reg, &ServerConfig{})
	require.NoError(t, err)

	err = ts.MountAndRun(func(router *mux.Router) {
		router.Methods("GET").Path("/hello").Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	})
	require.NoError(t, err)
	defer ts.StopAndShutdown()

	// test mounted endpoint
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/hello", ts.HTTPListenAddr()), nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	// assert all metrics have the prefix applied
	metrics, err := reg.Gather()
	require.NoError(t, err)
	for _, m := range metrics {
		require.True(t, strings.HasPrefix(m.GetName(), "test_namespace"))
	}
}

func TestTargetServer_ServerAPI(t *testing.T) {
	// dependencies
	reg := prometheus.NewRegistry()
	ts, err := NewTargetServer(util.TestLogger(t), "test_namespace", reg, &ServerConfig{
		EnableServerAPI: true,
	})
	require.NoError(t, err)

	err = ts.MountAndRun(func(router *mux.Router) {
		router.Methods("GET").Path("/hello").Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	})
	require.NoError(t, err)
	defer ts.StopAndShutdown()

	checkHealth := func(t *testing.T, expectedStatus int) {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/server/health", ts.HTTPListenAddr()), nil)
		require.NoError(t, err)
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, expectedStatus, res.StatusCode)
	}

	toggle := func(t *testing.T) {
		req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s/server/toggle_ready", ts.HTTPListenAddr()), nil)
		require.NoError(t, err)
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, res.StatusCode)
	}

	checkHealth(t, 200)

	// toggle to not ready
	toggle(t)
	checkHealth(t, 500)

	// toggle back to ready
	toggle(t)
	checkHealth(t, 200)
}

func TestTargetServer_NilConfig(t *testing.T) {
	reg := prometheus.NewRegistry()
	ts, err := NewTargetServer(util.TestLogger(t), "test_namespace", reg, nil)
	require.NoError(t, err)

	err = ts.MountAndRun(func(router *mux.Router) {})
	require.NoError(t, err)
	defer ts.StopAndShutdown()

	require.Equal(t, "[::]:8080", ts.HTTPListenAddr())
	// not asserting over grpc port since a random should have been assigned
}
