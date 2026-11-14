package cloudflare

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const (
	testAPIToken  = "test-token"
	testAccountID = "account"
	createdID     = "created"
	tunnelToken   = "tunnel-token"
)

type recordedRequest struct {
	method        string
	path          string
	query         string
	body          string
	authorization string
	contentType   string
}

type fakeResponse struct {
	status     int
	retryAfter string
	body       string
}

// fakeAPI records every request and answers with the queued responses in order, then with a
// successful empty envelope.
type fakeAPI struct {
	mutex     sync.Mutex
	requests  []recordedRequest
	responses []fakeResponse
}

func (f *fakeAPI) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	body, _ := io.ReadAll(request.Body)
	f.mutex.Lock()
	f.requests = append(f.requests, recordedRequest{
		method:        request.Method,
		path:          request.URL.Path,
		query:         request.URL.RawQuery,
		body:          string(body),
		authorization: request.Header.Get(authorizationHeader),
		contentType:   request.Header.Get(contentTypeHeader),
	})
	response := fakeResponse{status: http.StatusOK, body: `{"success":true,"errors":[],"messages":[],"result":{"id":"created"}}`}
	if len(f.responses) > 0 {
		response, f.responses = f.responses[0], f.responses[1:]
	}
	f.mutex.Unlock()

	writer.Header().Set(contentTypeHeader, jsonContentType)
	if response.retryAfter != "" {
		writer.Header().Set(retryAfterHeader, response.retryAfter)
	}
	writer.WriteHeader(response.status)
	_, _ = io.WriteString(writer, response.body)
}

func newTestClient(t *testing.T, responses ...fakeResponse) (*Client, *fakeAPI) {
	t.Helper()
	api := &fakeAPI{responses: responses}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)
	return newClient(server.URL+"/", testAPIToken, testAccountID), api
}

func ok(result string) fakeResponse {
	return fakeResponse{status: http.StatusOK, body: `{"success":true,"errors":[],"messages":[],"result":` + result + `}`}
}

var (
	fullPolicy = PolicyPayload{
		Name:     "policy",
		Decision: "allow",
		Include:  []Rule{{Everyone: true}, {Email: "user@example.com"}},
		Exclude:  []Rule{{EmailDomain: "example.com"}},
		Require:  []Rule{{IPRange: "10.0.0.0/8"}},
	}
	fullPolicyBody = `{"decision":"allow","exclude":[{"email_domain":{"domain":"example.com"}}],"include":[{"everyone":{}},{"email":{"email":"user@example.com"}}],"name":"policy","require":[{"ip":{"ip":"10.0.0.0/8"}}]}`

	fullApplication = ApplicationPayload{
		Name:            "application",
		Domain:          "app.example.com",
		Type:            "self_hosted",
		SessionDuration: "24h",
		Policies:        []string{"first", "second"},
	}
	fullApplicationBody = `{"domain":"app.example.com","name":"application","policies":["first","second"],"session_duration":"24h","type":"self_hosted"}`
)

// The expected requests below were recorded from cloudflare-go v7.11.0 against a fake server.
func TestRequestsMatchTheSDK(t *testing.T) {
	testCases := []struct {
		name     string
		call     func(ctx context.Context, client *Client) error
		expected recordedRequest
	}{
		{
			name: "create policy",
			call: func(ctx context.Context, client *Client) error {
				id, err := client.CreatePolicy(ctx, fullPolicy)
				return expectValue(id, createdID, err)
			},
			expected: recordedRequest{method: http.MethodPost, path: "/accounts/account/access/policies", body: fullPolicyBody},
		},
		{
			name: "create policy without exclude or require",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.CreatePolicy(ctx, PolicyPayload{Name: "policy", Decision: "deny", Include: []Rule{{Everyone: true}}})
				return err
			},
			expected: recordedRequest{method: http.MethodPost, path: "/accounts/account/access/policies", body: `{"decision":"deny","exclude":[],"include":[{"everyone":{}}],"name":"policy","require":[]}`},
		},
		{
			name: "update policy",
			call: func(ctx context.Context, client *Client) error {
				return client.UpdatePolicy(ctx, "policy-id", fullPolicy)
			},
			expected: recordedRequest{method: http.MethodPut, path: "/accounts/account/access/policies/policy-id", body: fullPolicyBody},
		},
		{
			name:     "delete policy",
			call:     func(ctx context.Context, client *Client) error { return client.DeletePolicy(ctx, "policy-id") },
			expected: recordedRequest{method: http.MethodDelete, path: "/accounts/account/access/policies/policy-id"},
		},
		{
			name: "create application",
			call: func(ctx context.Context, client *Client) error {
				id, err := client.CreateApplication(ctx, fullApplication)
				return expectValue(id, createdID, err)
			},
			expected: recordedRequest{method: http.MethodPost, path: "/accounts/account/access/apps", body: fullApplicationBody},
		},
		{
			name: "create application without policies or session duration",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.CreateApplication(ctx, ApplicationPayload{Name: "application", Domain: "app.example.com", Type: "self_hosted"})
				return err
			},
			expected: recordedRequest{method: http.MethodPost, path: "/accounts/account/access/apps", body: `{"domain":"app.example.com","name":"application","policies":[],"session_duration":"","type":"self_hosted"}`},
		},
		{
			name: "update application",
			call: func(ctx context.Context, client *Client) error {
				return client.UpdateApplication(ctx, "app-id", fullApplication)
			},
			expected: recordedRequest{method: http.MethodPut, path: "/accounts/account/access/apps/app-id", body: fullApplicationBody},
		},
		{
			name:     "delete application",
			call:     func(ctx context.Context, client *Client) error { return client.DeleteApplication(ctx, "app-id") },
			expected: recordedRequest{method: http.MethodDelete, path: "/accounts/account/access/apps/app-id"},
		},
		{
			name: "create tunnel",
			call: func(ctx context.Context, client *Client) error {
				id, err := client.CreateTunnel(ctx, "tunnel")
				return expectValue(id, createdID, err)
			},
			expected: recordedRequest{method: http.MethodPost, path: "/accounts/account/cfd_tunnel", body: `{"config_src":"cloudflare","name":"tunnel"}`},
		},
		{
			name:     "delete tunnel",
			call:     func(ctx context.Context, client *Client) error { return client.DeleteTunnel(ctx, "tunnel-id") },
			expected: recordedRequest{method: http.MethodDelete, path: "/accounts/account/cfd_tunnel/tunnel-id"},
		},
		{
			name: "create route",
			call: func(ctx context.Context, client *Client) error {
				return client.CreateRoute(ctx, "tunnel-id", "10.0.0.0/8")
			},
			expected: recordedRequest{method: http.MethodPost, path: "/accounts/account/teamnet/routes", body: `{"network":"10.0.0.0/8","tunnel_id":"tunnel-id"}`},
		},
		{
			name:     "delete route",
			call:     func(ctx context.Context, client *Client) error { return client.DeleteRoute(ctx, "route-id") },
			expected: recordedRequest{method: http.MethodDelete, path: "/accounts/account/teamnet/routes/route-id"},
		},
		{
			name: "put tunnel configuration",
			call: func(ctx context.Context, client *Client) error {
				return client.PutTunnelConfiguration(ctx, "tunnel-id", []IngressRule{
					{Hostname: "app.example.com", Service: "https://service", OriginRequest: &OriginRequest{MatchSNIToHost: true}},
					{Hostname: "other.example.com", Service: "http://other", OriginRequest: &OriginRequest{}},
					{Service: "http_status:404"},
				})
			},
			expected: recordedRequest{method: http.MethodPut, path: "/accounts/account/cfd_tunnel/tunnel-id/configurations", body: `{"config":{"ingress":[{"hostname":"app.example.com","originRequest":{"matchSNItoHost":true},"service":"https://service"},{"hostname":"other.example.com","originRequest":{"matchSNItoHost":false},"service":"http://other"},{"service":"http_status:404"}]}}`},
		},
		{
			name: "put empty tunnel configuration",
			call: func(ctx context.Context, client *Client) error {
				return client.PutTunnelConfiguration(ctx, "tunnel-id", nil)
			},
			expected: recordedRequest{method: http.MethodPut, path: "/accounts/account/cfd_tunnel/tunnel-id/configurations", body: `{"config":{"ingress":[]}}`},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, api := newTestClient(t)
			if err := testCase.call(context.Background(), client); err != nil {
				t.Fatal(err)
			}
			expected := testCase.expected
			expected.authorization = bearerPrefix + testAPIToken
			if expected.body != "" {
				expected.contentType = jsonContentType
			}
			if len(api.requests) != 1 || !reflect.DeepEqual(api.requests[0], expected) {
				t.Fatalf("requests %+v, expected %+v", api.requests, expected)
			}
		})
	}
}

func expectValue(actual, expected string, err error) error {
	if err != nil {
		return err
	}
	if actual != expected {
		return errors.New("got " + actual + ", expected " + expected)
	}
	return nil
}

func TestTunnelToken(t *testing.T) {
	client, api := newTestClient(t, ok(`"`+tunnelToken+`"`))
	token, err := client.TunnelToken(context.Background(), "tunnel-id")
	if err := expectValue(token, tunnelToken, err); err != nil {
		t.Fatal(err)
	}
	if api.requests[0].method != http.MethodGet || api.requests[0].path != "/accounts/account/cfd_tunnel/tunnel-id/token" {
		t.Fatalf("request %+v", api.requests[0])
	}
}

func TestListRoutesReadsEveryPage(t *testing.T) {
	client, api := newTestClient(t,
		ok(`[{"id":"first","network":"10.0.0.0/8"}]`),
		ok(`[{"id":"second","network":"10.1.0.0/16"},{"id":"third","network":"10.2.0.0/16"}]`),
		ok(`[]`),
	)
	routes, err := client.ListRoutes(context.Background(), "tunnel-id")
	if err != nil {
		t.Fatal(err)
	}
	expectedRoutes := []Route{{ID: "first", Network: "10.0.0.0/8"}, {ID: "second", Network: "10.1.0.0/16"}, {ID: "third", Network: "10.2.0.0/16"}}
	if !reflect.DeepEqual(routes, expectedRoutes) {
		t.Fatalf("routes %+v", routes)
	}
	var queries []string
	for _, request := range api.requests {
		if request.method != http.MethodGet || request.path != "/accounts/account/teamnet/routes" {
			t.Fatalf("request %+v", request)
		}
		queries = append(queries, request.query)
	}
	expectedQueries := []string{"page=1&tunnel_id=tunnel-id", "page=2&tunnel_id=tunnel-id", "page=3&tunnel_id=tunnel-id"}
	if !reflect.DeepEqual(queries, expectedQueries) {
		t.Fatalf("queries %v", queries)
	}
}

func TestErrorIncludesCloudflareMessages(t *testing.T) {
	client, _ := newTestClient(t, fakeResponse{status: http.StatusBadRequest, body: `{"success":false,"errors":[{"code":1000,"message":"bad request"}],"messages":[],"result":null}`})
	err := client.DeletePolicy(context.Background(), "policy-id")
	if err == nil || !strings.Contains(err.Error(), "400 Bad Request: bad request (1000)") {
		t.Fatalf("error %v", err)
	}
	var apiError *requestError
	if !errors.As(err, &apiError) || apiError.statusCode != http.StatusBadRequest {
		t.Fatalf("error %#v", err)
	}
}

func TestUnsuccessfulEnvelopeIsAnError(t *testing.T) {
	client, _ := newTestClient(t, fakeResponse{status: http.StatusOK, body: `{"success":false,"errors":[{"code":1001,"message":"rejected"}],"result":null}`})
	if _, err := client.CreateTunnel(context.Background(), "tunnel"); err == nil || !strings.Contains(err.Error(), "rejected (1001)") {
		t.Fatalf("error %v", err)
	}
}

func TestRetriesRateLimitsAndServerErrors(t *testing.T) {
	client, api := newTestClient(t,
		fakeResponse{status: http.StatusTooManyRequests, retryAfter: "0", body: `{"success":false,"errors":[]}`},
		fakeResponse{status: http.StatusBadGateway, retryAfter: "0", body: `{"success":false,"errors":[]}`},
		ok(`{"id":"created"}`),
	)
	id, err := client.CreateTunnel(context.Background(), "tunnel")
	if err := expectValue(id, createdID, err); err != nil {
		t.Fatal(err)
	}
	if len(api.requests) != maximumRetries+1 {
		t.Fatalf("got %d requests", len(api.requests))
	}
	for _, request := range api.requests {
		if request.body != `{"config_src":"cloudflare","name":"tunnel"}` {
			t.Fatalf("retried body %q", request.body)
		}
	}
}

func TestStopsAfterMaximumRetries(t *testing.T) {
	unavailable := fakeResponse{status: http.StatusServiceUnavailable, retryAfter: "0", body: `{"success":false,"errors":[{"code":1,"message":"down"}]}`}
	client, api := newTestClient(t, unavailable, unavailable, unavailable, ok(`{"id":"created"}`))
	if err := client.DeleteTunnel(context.Background(), "tunnel-id"); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("error %v", err)
	}
	if len(api.requests) != maximumRetries+1 {
		t.Fatalf("got %d requests", len(api.requests))
	}
}

func TestDoesNotRetryClientErrors(t *testing.T) {
	client, api := newTestClient(t, fakeResponse{status: http.StatusForbidden, body: `{"success":false,"errors":[]}`})
	if err := client.DeleteRoute(context.Background(), "route-id"); err == nil {
		t.Fatal("expected an error")
	}
	if len(api.requests) != 1 {
		t.Fatalf("got %d requests", len(api.requests))
	}
}
