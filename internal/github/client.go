package github

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const apiBase = "https://api.github.com"

type Client struct {
	httpClient   *http.Client
	token        string
	requestLimit chan struct{}
}

func NewClient(
	token string,
	workers int,
) *Client {
	if workers < 1 {
		workers = 1
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,

		// This is the hard limit on simultaneous requests to the Github API.
		requestLimit: make(chan struct{}, workers),
	}
}

func (c *Client) Get(
	path string,
	result any,
) error {
	c.requestLimit <- struct{}{}

	defer func() {
		<-c.requestLimit
	}()

	req, err := http.NewRequest(
		http.MethodGet,
		apiBase+path,
		nil,
	)
	if err != nil {
		return fmt.Errorf(
			"creating Github request: %w",
			err,
		)
	}

	slog.Debug(
		"Github API request",
		"method", req.Method,
		"path", path,
	)

	req.Header.Set(
		"Accept",
		"application/vnd.github+json",
	)

	req.Header.Set(
		"X-Github-Api-Version",
		"2022-11-28",
	)

	if c.token != "" {
		req.Header.Set(
			"Authorization",
			"Bearer "+c.token,
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf(
			"Github request failed: %w",
			err,
		)
	}

	defer resp.Body.Close()

	slog.Debug(
		"Github API response",
		"status", resp.StatusCode,
		"path", path,
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf(
			"Github API returned %s for %s: %s",
			resp.Status,
			req.URL.String(),
			strings.TrimSpace(string(body)),
		)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf(
			"decoding Github response: %w",
			err,
		)
	}

	return nil
}
