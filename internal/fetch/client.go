package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	httpClient      *http.Client
	proxyHttpClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	if proxy := os.Getenv("FETCH_PROXY"); proxy != "" {
		if u, err := url.Parse(proxy); err == nil {
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.Proxy = http.ProxyURL(u)
			c.proxyHttpClient = &http.Client{
				Timeout:   timeout,
				Transport: transport,
			}
		}
	}

	return c
}

type Response struct {
	Body    []byte
	Headers http.Header
}

func (c *Client) Fetch(ctx context.Context, url string) (Response, error) {
	// First attempt without proxy
	res, err := c.doFetch(ctx, c.httpClient, url)
	if err == nil {
		return res, nil
	}

	// Second attempt with proxy if configured
	if c.proxyHttpClient != nil {
		return c.doFetch(ctx, c.proxyHttpClient, url)
	}

	return Response{}, err
}

func (c *Client) doFetch(ctx context.Context, httpClient *http.Client, url string) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("User-Agent", "clash-verge/1.4.11")
	resp, err := httpClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	return Response{Body: body, Headers: resp.Header.Clone()}, nil
}

type SubscriptionInfo struct {
	Used   int64
	Total  int64
	Expire int64
}

func ParseSubscriptionUserinfo(raw string) (SubscriptionInfo, bool) {
	var upload, download int64
	var meta SubscriptionInfo
	parts := strings.Split(raw, ";")
	for _, part := range parts {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			return SubscriptionInfo{}, false
		}
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return SubscriptionInfo{}, false
		}
		switch strings.TrimSpace(key) {
		case "upload":
			upload = n
		case "download":
			download = n
		case "total":
			meta.Total = n
		case "expire":
			meta.Expire = n
		}
	}
	meta.Used = upload + download
	return meta, meta.Total > 0 || meta.Expire > 0 || meta.Used > 0
}
