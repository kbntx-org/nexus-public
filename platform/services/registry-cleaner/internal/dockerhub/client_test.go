package dockerhub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testNamespace = "kbntx"
	testUsername  = "robot"
	testToken     = "personal-access-token"
)

type fakeHub struct {
	server         *httptest.Server
	tokensIssued   atomic.Int32
	deleted        []string
	rejectNextCall atomic.Bool
	rateLimitNext  atomic.Bool
}

func newFakeHub(t *testing.T) *fakeHub {
	hub := &fakeHub{}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v2/auth/token", func(writer http.ResponseWriter, request *http.Request) {
		var body tokenRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.Identifier != testUsername || body.Secret != testToken {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		issued := hub.tokensIssued.Add(1)
		writeJSON(t, writer, tokenResponse{AccessToken: fmt.Sprintf("access-%d", issued)})
	})

	mux.HandleFunc("GET /v2/namespaces/kbntx/repositories", func(writer http.ResponseWriter, request *http.Request) {
		if !hub.authorized(writer, request) {
			return
		}
		if request.URL.Query().Get("page") == "" {
			writeJSON(t, writer, map[string]any{
				"next":    hub.server.URL + "/v2/namespaces/kbntx/repositories?page=2&page_size=100",
				"results": []map[string]any{{"name": "smelt"}},
			})
			return
		}
		writeJSON(t, writer, map[string]any{"next": nil, "results": []map[string]any{{"name": "portfolio"}}})
	})

	mux.HandleFunc("GET /v2/namespaces/kbntx/repositories/smelt/tags", func(writer http.ResponseWriter, request *http.Request) {
		if !hub.authorized(writer, request) {
			return
		}
		pushed := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
		if request.URL.Query().Get("page") == "" {
			writeJSON(t, writer, map[string]any{
				"next": hub.server.URL + "/v2/namespaces/kbntx/repositories/smelt/tags?page=2&page_size=100",
				"results": []map[string]any{
					{"name": "first", "digest": "sha256:first", "tag_last_pushed": pushed, "last_updated": pushed},
				},
			})
			return
		}
		writeJSON(t, writer, map[string]any{
			"next": nil,
			"results": []map[string]any{
				{"name": "never-pushed", "digest": "sha256:second", "tag_last_pushed": nil, "last_updated": pushed},
				{"name": "no-dates", "digest": "sha256:third", "tag_last_pushed": nil, "last_updated": nil},
			},
		})
	})

	mux.HandleFunc("DELETE /v2/namespaces/kbntx/repositories/smelt/tags/{tag}", func(writer http.ResponseWriter, request *http.Request) {
		if !hub.authorized(writer, request) {
			return
		}
		if request.PathValue("tag") == "missing" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if request.PathValue("tag") == "protected" {
			writer.WriteHeader(http.StatusForbidden)
			return
		}
		hub.deleted = append(hub.deleted, request.PathValue("tag"))
		writer.WriteHeader(http.StatusNoContent)
	})

	hub.server = httptest.NewServer(mux)
	t.Cleanup(hub.server.Close)
	return hub
}

func (h *fakeHub) authorized(writer http.ResponseWriter, request *http.Request) bool {
	if h.rateLimitNext.CompareAndSwap(true, false) {
		writer.Header().Set(retryAfterHeader, "7")
		writer.WriteHeader(http.StatusTooManyRequests)
		return false
	}
	if h.rejectNextCall.CompareAndSwap(true, false) {
		writer.WriteHeader(http.StatusUnauthorized)
		return false
	}
	if request.Header.Get(authorizationHeader) != fmt.Sprintf("%saccess-%d", bearerPrefix, h.tokensIssued.Load()) {
		writer.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func writeJSON(t *testing.T, writer http.ResponseWriter, body any) {
	writer.Header().Set(contentTypeHeader, jsonContentType)
	if err := json.NewEncoder(writer).Encode(body); err != nil {
		t.Errorf("encode: %v", err)
	}
}

func newTestClient(hub *fakeHub) (*Client, *[]time.Duration) {
	client := NewClient(hub.server.URL, testNamespace, testUsername, testToken)
	var waits []time.Duration
	client.wait = func(_ context.Context, duration time.Duration) error {
		waits = append(waits, duration)
		return nil
	}
	return client, &waits
}

func TestListTagsFollowsPagesAndFallsBackOnPushDate(t *testing.T) {
	hub := newFakeHub(t)
	client, _ := newTestClient(hub)

	tags, err := client.ListTags(context.Background(), "smelt")
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("got %d tags, expected 3", len(tags))
	}
	if tags[0].Name != "first" || tags[0].PushedAt.IsZero() {
		t.Errorf("unexpected first tag %+v", tags[0])
	}
	if tags[1].PushedAt.IsZero() {
		t.Error("expected last_updated to stand in for a missing tag_last_pushed")
	}
	if !tags[2].PushedAt.IsZero() {
		t.Errorf("expected an unknown push date, got %v", tags[2].PushedAt)
	}
}

func TestListRepositoriesFollowsPages(t *testing.T) {
	hub := newFakeHub(t)
	client, _ := newTestClient(hub)

	repositories, err := client.ListRepositories(context.Background())
	if err != nil {
		t.Fatalf("list repositories: %v", err)
	}
	if !slices.Equal(repositories, []string{"smelt", "portfolio"}) {
		t.Errorf("got %v, expected [smelt portfolio]", repositories)
	}
}

func TestDeleteTag(t *testing.T) {
	hub := newFakeHub(t)
	client, _ := newTestClient(hub)
	ctx := context.Background()

	if err := client.DeleteTag(ctx, "smelt", "old"); err != nil {
		t.Errorf("delete old: %v", err)
	}
	if err := client.DeleteTag(ctx, "smelt", "missing"); err != nil {
		t.Errorf("an already deleted tag should not fail: %v", err)
	}
	if err := client.DeleteTag(ctx, "smelt", "protected"); err == nil {
		t.Error("expected a forbidden delete to fail")
	}
	if !slices.Equal(hub.deleted, []string{"old"}) {
		t.Errorf("deleted %v, expected [old]", hub.deleted)
	}
}

func TestSendRenewsAnExpiredToken(t *testing.T) {
	hub := newFakeHub(t)
	client, _ := newTestClient(hub)
	ctx := context.Background()

	if err := client.DeleteTag(ctx, "smelt", "first"); err != nil {
		t.Fatalf("delete first: %v", err)
	}
	hub.rejectNextCall.Store(true)
	if err := client.DeleteTag(ctx, "smelt", "second"); err != nil {
		t.Fatalf("delete after expiry: %v", err)
	}
	if issued := hub.tokensIssued.Load(); issued != 2 {
		t.Errorf("issued %d tokens, expected 2", issued)
	}
}

func TestSendWaitsOutRateLimits(t *testing.T) {
	hub := newFakeHub(t)
	client, waits := newTestClient(hub)
	hub.rateLimitNext.Store(true)

	if err := client.DeleteTag(context.Background(), "smelt", "old"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !slices.Equal(*waits, []time.Duration{7 * time.Second}) {
		t.Errorf("waited %v, expected [7s]", *waits)
	}
}

func TestAuthenticateRejectsBadCredentials(t *testing.T) {
	hub := newFakeHub(t)
	client := NewClient(hub.server.URL, testNamespace, testUsername, "wrong")
	if _, err := client.ListTags(context.Background(), "smelt"); err == nil {
		t.Error("expected bad credentials to fail")
	}
}
