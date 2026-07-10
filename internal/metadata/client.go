package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type HTTPClient struct {
	base   string
	client *http.Client
	logger *slog.Logger
}

func NewClient(base string, client *http.Client, logger *slog.Logger) *HTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPClient{base: strings.TrimRight(base, "/"), client: client, logger: logger}
}

func (c *HTTPClient) Task(ctx context.Context) (Task, error) {
	if c.base == "" {
		return Task{}, errors.New("ECS_CONTAINER_METADATA_URI_V4 is empty")
	}
	var out Task
	if err := c.get(ctx, c.base+"/task", &out); err != nil {
		return Task{}, err
	}
	return out, nil
}

func (c *HTTPClient) TaskStats(ctx context.Context) (TaskStats, error) {
	if c.base == "" {
		return nil, errors.New("ECS_CONTAINER_METADATA_URI_V4 is empty")
	}
	var out TaskStats
	if err := c.get(ctx, c.base+"/task/stats", &out); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for k, v := range out {
		v.Timestamp = now
		out[k] = v
	}
	return out, nil
}

func (c *HTTPClient) get(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("metadata endpoint %s returned %s", url, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}
