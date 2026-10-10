package dockerhub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kbntx-org/nexus/platform/services/registry-cleaner/internal/retention"
)

const (
	DefaultBaseURL = "https://hub.docker.com"

	pageSize = 100

	maxAttempts           = 5
	defaultRetryAfter     = time.Minute
	requestTimeout        = 30 * time.Second
	errorBodyLimitInBytes = 512
	retryAfterHeader      = "Retry-After"
	authorizationHeader   = "Authorization"
	contentTypeHeader     = "Content-Type"
	jsonContentType       = "application/json"
	bearerPrefix          = "Bearer "
)

var errUnexpectedStatus = errors.New("unexpected status")

type Client struct {
	httpClient  *http.Client
	baseURL     string
	namespace   string
	username    string
	token       string
	accessToken string
	wait        func(ctx context.Context, duration time.Duration) error
}

func NewClient(baseURL string, namespace string, username string, token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: requestTimeout},
		baseURL:    baseURL,
		namespace:  namespace,
		username:   username,
		token:      token,
		wait:       waitFor,
	}
}

type page[Result any] struct {
	Next    string   `json:"next"`
	Results []Result `json:"results"`
}

type repositoryResult struct {
	Name string `json:"name"`
}

type tagResult struct {
	Name          string     `json:"name"`
	TagLastPushed *time.Time `json:"tag_last_pushed"`
	LastUpdated   *time.Time `json:"last_updated"`
}

func (c *Client) ListRepositories(ctx context.Context) ([]string, error) {
	repositoriesURL := fmt.Sprintf("%s/v2/namespaces/%s/repositories?page_size=%d",
		c.baseURL, url.PathEscape(c.namespace), pageSize)
	results, err := listAll[repositoryResult](ctx, c, repositoriesURL)
	if err != nil {
		return nil, err
	}
	repositories := make([]string, 0, len(results))
	for _, result := range results {
		repositories = append(repositories, result.Name)
	}
	return repositories, nil
}

func (c *Client) ListTags(ctx context.Context, repository string) ([]retention.Tag, error) {
	tagsURL := fmt.Sprintf("%s/v2/namespaces/%s/repositories/%s/tags?page_size=%d",
		c.baseURL, url.PathEscape(c.namespace), url.PathEscape(repository), pageSize)
	results, err := listAll[tagResult](ctx, c, tagsURL)
	if err != nil {
		return nil, err
	}
	tags := make([]retention.Tag, 0, len(results))
	for _, result := range results {
		tag := retention.Tag{Name: result.Name}
		switch {
		case result.TagLastPushed != nil:
			tag.PushedAt = *result.TagLastPushed
		case result.LastUpdated != nil:
			tag.PushedAt = *result.LastUpdated
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func listAll[Result any](ctx context.Context, c *Client, pageURL string) ([]Result, error) {
	var results []Result
	for pageURL != "" {
		response, err := c.send(ctx, http.MethodGet, pageURL)
		if err != nil {
			return nil, err
		}
		var current page[Result]
		if err := decodeResponse(response, http.StatusOK, &current); err != nil {
			return nil, err
		}
		results = append(results, current.Results...)
		pageURL = current.Next
	}
	return results, nil
}

func (c *Client) DeleteTag(ctx context.Context, repository string, tag string) error {
	tagURL := fmt.Sprintf("%s/v2/namespaces/%s/repositories/%s/tags/%s",
		c.baseURL, url.PathEscape(c.namespace), url.PathEscape(repository), url.PathEscape(tag))
	response, err := c.send(ctx, http.MethodDelete, tagURL)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	switch response.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound:
		return nil
	default:
		return statusError(response)
	}
}

type tokenRequest struct {
	Identifier string `json:"identifier"`
	Secret     string `json:"secret"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (c *Client) authenticate(ctx context.Context) error {
	body, err := json.Marshal(tokenRequest{Identifier: c.username, Secret: c.token})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/auth/token", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set(contentTypeHeader, jsonContentType)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	var token tokenResponse
	if err := decodeResponse(response, http.StatusOK, &token); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	c.accessToken = token.AccessToken
	return nil
}

func (c *Client) send(ctx context.Context, method string, requestURL string) (*http.Response, error) {
	if c.accessToken == "" {
		if err := c.authenticate(ctx); err != nil {
			return nil, err
		}
	}
	renewed := false
	for attempt := 1; ; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set(authorizationHeader, bearerPrefix+c.accessToken)
		response, err := c.httpClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", method, requestURL, err)
		}
		switch {
		case response.StatusCode == http.StatusUnauthorized && !renewed:
			_ = response.Body.Close()
			renewed = true
			if err := c.authenticate(ctx); err != nil {
				return nil, err
			}
		case response.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts:
			_ = response.Body.Close()
			if err := c.wait(ctx, retryAfter(response)); err != nil {
				return nil, err
			}
		default:
			return response, nil
		}
	}
}

func retryAfter(response *http.Response) time.Duration {
	seconds, err := strconv.Atoi(response.Header.Get(retryAfterHeader))
	if err != nil || seconds <= 0 {
		return defaultRetryAfter
	}
	return time.Duration(seconds) * time.Second
}

func waitFor(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func decodeResponse(response *http.Response, expectedStatus int, target any) error {
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != expectedStatus {
		return statusError(response)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", response.Request.URL, err)
	}
	return nil
}

func statusError(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, errorBodyLimitInBytes))
	return fmt.Errorf("%s %s: %w %d: %s",
		response.Request.Method, response.Request.URL, errUnexpectedStatus, response.StatusCode, bytes.TrimSpace(body))
}
