package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.cloudflare.com/client/v4/"

	contentTypeHeader   = "Content-Type"
	acceptHeader        = "Accept"
	authorizationHeader = "Authorization"
	retryAfterHeader    = "Retry-After"
	jsonContentType     = "application/json"
	bearerPrefix        = "Bearer "

	pageQueryParameter = "page"
	firstPage          = 1

	maximumRetries    = 2
	initialRetryDelay = 500 * time.Millisecond
	maximumRetryDelay = 8 * time.Second
	retryJitterShare  = 4
)

type apiClient struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string
}

type envelope struct {
	Success bool            `json:"success"`
	Errors  []responseError `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type requestError struct {
	method     string
	path       string
	statusCode int
	errors     []responseError
}

func (e *requestError) Error() string {
	messages := make([]string, len(e.errors))
	for index, responseError := range e.errors {
		messages[index] = fmt.Sprintf("%s (%d)", responseError.Message, responseError.Code)
	}
	return fmt.Sprintf("%s %s: %d %s: %s", e.method, e.path, e.statusCode, http.StatusText(e.statusCode), strings.Join(messages, "; "))
}

// do sends one API request and decodes the envelope's result into result when it is not nil,
// retrying like the Cloudflare SDK did: twice, on connection errors, 408, 409, 429 and 5xx.
func (a *apiClient) do(ctx context.Context, method, path string, query url.Values, body, result any) error {
	var encodedBody []byte
	if body != nil {
		var err error
		if encodedBody, err = json.Marshal(body); err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
	}

	for attempt := 0; ; attempt++ {
		response, err := a.send(ctx, method, path, query, encodedBody)
		if attempt == maximumRetries || !shouldRetry(ctx, response, err) {
			if err != nil {
				return err
			}
			return decode(method, path, response, result)
		}
		delay := retryDelay(response, attempt)
		if response != nil {
			_ = response.Body.Close()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

func (a *apiClient) send(ctx context.Context, method, path string, query url.Values, encodedBody []byte) (*http.Response, error) {
	address := a.baseURL + path
	if len(query) > 0 {
		address += "?" + query.Encode()
	}
	var bodyReader io.Reader
	if encodedBody != nil {
		bodyReader = bytes.NewReader(encodedBody)
	}
	request, err := http.NewRequestWithContext(ctx, method, address, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request %s %s: %w", method, path, err)
	}
	request.Header.Set(authorizationHeader, bearerPrefix+a.apiToken)
	request.Header.Set(acceptHeader, jsonContentType)
	if encodedBody != nil {
		request.Header.Set(contentTypeHeader, jsonContentType)
	}
	response, err := a.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request %s %s: %w", method, path, err)
	}
	return response, nil
}

func shouldRetry(ctx context.Context, response *http.Response, err error) bool {
	if err != nil {
		return ctx.Err() == nil
	}
	return response.StatusCode == http.StatusRequestTimeout ||
		response.StatusCode == http.StatusConflict ||
		response.StatusCode == http.StatusTooManyRequests ||
		response.StatusCode >= http.StatusInternalServerError
}

func retryDelay(response *http.Response, attempt int) time.Duration {
	if response != nil {
		if seconds, err := strconv.ParseFloat(response.Header.Get(retryAfterHeader), 64); err == nil {
			return max(0, time.Duration(seconds*float64(time.Second)))
		}
	}
	delay := min(initialRetryDelay<<attempt, maximumRetryDelay)
	return delay - rand.N(delay/retryJitterShare)
}

func decode(method, path string, response *http.Response, result any) error {
	defer func() { _ = response.Body.Close() }()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response %s %s: %w", method, path, err)
	}

	var decoded envelope
	decodeErr := json.Unmarshal(contents, &decoded)
	if response.StatusCode >= http.StatusBadRequest || (decodeErr == nil && !decoded.Success) {
		return &requestError{method: method, path: path, statusCode: response.StatusCode, errors: decoded.Errors}
	}
	if decodeErr != nil {
		return fmt.Errorf("decode response %s %s: %w", method, path, decodeErr)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(decoded.Result, result); err != nil {
		return fmt.Errorf("decode result %s %s: %w", method, path, err)
	}
	return nil
}

// listAllPages requests pages until one comes back empty, as the SDK's auto-paging did,
// rather than trusting result_info's counts.
func listAllPages[Item any](ctx context.Context, api *apiClient, path string, query url.Values) ([]Item, error) {
	var items []Item
	for page := firstPage; ; page++ {
		pageQuery := url.Values{}
		for key, values := range query {
			pageQuery[key] = values
		}
		pageQuery.Set(pageQueryParameter, strconv.Itoa(page))

		var pageItems []Item
		if err := api.do(ctx, http.MethodGet, path, pageQuery, nil, &pageItems); err != nil {
			return nil, err
		}
		if len(pageItems) == 0 {
			return items, nil
		}
		items = append(items, pageItems...)
	}
}
