package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFetchRetry(t *testing.T) {
	var count int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "clash-verge/1.4.11", r.Header.Get("User-Agent"))
		atomic.AddInt32(&count, 1)
		if atomic.LoadInt32(&count) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	// Mock proxy that just forwards to the test server
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "clash-verge/1.4.11", r.Header.Get("User-Agent"))
		if atomic.LoadInt32(&count) == 1 {
			atomic.AddInt32(&count, 1)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}
	}))
	defer proxy.Close()

	os.Setenv("FETCH_PROXY", proxy.URL)
	defer os.Unsetenv("FETCH_PROXY")

	client := NewClient(5 * time.Second)
	res, err := client.Fetch(context.Background(), ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, "ok", string(res.Body))
	// 1 direct (failed) + 1 via proxy (success)
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestFetchRetryFailure(t *testing.T) {
	var count int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()

	// Mock proxy that also fails
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()

	os.Setenv("FETCH_PROXY", proxy.URL)
	defer os.Unsetenv("FETCH_PROXY")

	client := NewClient(5 * time.Second)
	_, err := client.Fetch(context.Background(), ts.URL)
	assert.Error(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestFetchProxyEnv(t *testing.T) {
	os.Setenv("FETCH_PROXY", "http://localhost:1234")
	defer os.Unsetenv("FETCH_PROXY")

	client := NewClient(5 * time.Second)
	assert.NotNil(t, client.proxyHttpClient)
	
	// Default client should NOT have proxy
	if client.httpClient.Transport != nil {
		transport := client.httpClient.Transport.(*http.Transport)
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		proxyURL, err := transport.Proxy(req)
		assert.NoError(t, err)
		assert.Nil(t, proxyURL)
	}

	// Proxy client SHOULD have proxy
	transport := client.proxyHttpClient.Transport.(*http.Transport)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	proxyURL, err := transport.Proxy(req)
	assert.NoError(t, err)
	assert.NotNil(t, proxyURL)
	assert.Equal(t, "localhost:1234", proxyURL.Host)
}
