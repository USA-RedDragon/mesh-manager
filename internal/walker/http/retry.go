package http

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
)

type Client struct {
	client  http.Client
	retries int
	jitter  time.Duration
}

func NewClient(timeout time.Duration, retries int, jitter time.Duration) *Client {
	return &Client{
		client: http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if req.URL.Path == "/a/sysinfo" {
					return nil
				}
				return http.ErrUseLastResponse
			},
		},
		retries: retries,
		jitter:  jitter,
	}
}

func (c *Client) jitterSleep() {
	//nolint:gosec
	time.Sleep(time.Duration(rand.Int63n(int64(c.jitter))))
}

func (c *Client) Get(ctx context.Context, url string) (*apimodels.SysinfoResponse, error) {
	var resp *http.Response
	c.jitterSleep()

	for n := range c.retries {
		var err error
		resp, err = c.get(ctx, url)
		if err != nil {
			if n == c.retries-1 {
				return nil, fmt.Errorf("failed to get url after %d retries: %w", c.retries, err)
			}
			c.jitterSleep()
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			if n == c.retries-1 {
				return nil, fmt.Errorf("received non-200 status code after %d retries", c.retries)
			}
			c.jitterSleep()
			continue
		}
		break
	}

	var response apimodels.SysinfoResponse
	if err := response.Decode(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return &response, nil
}

func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get error: %w", err)
	}

	return resp, nil
}
