package http

import (
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
		},
		retries: retries,
		jitter:  jitter,
	}
}

func (c *Client) jitterSleep() {
	//nolint:golint,gosec
	time.Sleep(time.Duration(rand.Int63n(int64(c.jitter))))
}

func (c *Client) Get(url string) (*apimodels.SysinfoResponse, error) {
	var resp *http.Response
	c.jitterSleep()

	for n := range c.retries {
		var err error
		resp, err = c.get(url)
		if err != nil {
			if n == c.retries-1 {
				return nil, err
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
		return nil, err
	}

	return &response, nil
}

func (c *Client) get(url string) (*http.Response, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
