package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

const (
	policiesPath            = "accounts/%s/access/policies"
	policyPath              = "accounts/%s/access/policies/%s"
	applicationsPath        = "accounts/%s/access/apps"
	applicationPath         = "accounts/%s/access/apps/%s"
	tunnelsPath             = "accounts/%s/cfd_tunnel"
	tunnelPath              = "accounts/%s/cfd_tunnel/%s"
	tunnelTokenPath         = "accounts/%s/cfd_tunnel/%s/token"
	tunnelConfigurationPath = "accounts/%s/cfd_tunnel/%s/configurations"
	routesPath              = "accounts/%s/teamnet/routes"
	routePath               = "accounts/%s/teamnet/routes/%s"

	tunnelConfigSourceCloudflare = "cloudflare"
	tunnelIDQueryParameter       = "tunnel_id"
)

type Client struct {
	api       *apiClient
	accountID string
}

func NewClient(apiToken, accountID string) *Client {
	return newClient(defaultBaseURL, apiToken, accountID)
}

func newClient(baseURL, apiToken, accountID string) *Client {
	return &Client{
		api:       &apiClient{httpClient: http.DefaultClient, baseURL: baseURL, apiToken: apiToken},
		accountID: accountID,
	}
}

func (c *Client) accountPath(format string, identifiers ...string) string {
	segments := []any{url.PathEscape(c.accountID)}
	for _, identifier := range identifiers {
		segments = append(segments, url.PathEscape(identifier))
	}
	return fmt.Sprintf(format, segments...)
}

type Rule struct {
	Everyone    bool
	Email       string
	EmailDomain string
	IPRange     string
}

type PolicyPayload struct {
	Name     string
	Decision string
	Include  []Rule
	Exclude  []Rule
	Require  []Rule
}

type ApplicationPayload struct {
	Name            string
	Domain          string
	Type            string
	SessionDuration string
	Policies        []string
}

type emptyRule struct{}

type emailRule struct {
	Email string `json:"email"`
}

type emailDomainRule struct {
	Domain string `json:"domain"`
}

type ipRule struct {
	IP string `json:"ip"`
}

type accessRule struct {
	Email       *emailRule       `json:"email,omitempty"`
	EmailDomain *emailDomainRule `json:"email_domain,omitempty"`
	Everyone    *emptyRule       `json:"everyone,omitempty"`
	IP          *ipRule          `json:"ip,omitempty"`
}

// Fields are alphabetical, matching the Cloudflare SDK's request bodies that the tests pin.
type policyBody struct {
	Decision string       `json:"decision"`
	Exclude  []accessRule `json:"exclude"`
	Include  []accessRule `json:"include"`
	Name     string       `json:"name"`
	Require  []accessRule `json:"require"`
}

type applicationBody struct {
	Domain          string   `json:"domain"`
	Name            string   `json:"name"`
	Policies        []string `json:"policies"`
	SessionDuration string   `json:"session_duration"`
	Type            string   `json:"type"`
}

type createdResource struct {
	ID string `json:"id"`
}

func toAccessRules(rules []Rule) []accessRule {
	converted := make([]accessRule, len(rules))
	for index, rule := range rules {
		converted[index] = toAccessRule(rule)
	}
	return converted
}

func toAccessRule(rule Rule) accessRule {
	switch {
	case rule.Everyone:
		return accessRule{Everyone: &emptyRule{}}
	case rule.Email != "":
		return accessRule{Email: &emailRule{Email: rule.Email}}
	case rule.EmailDomain != "":
		return accessRule{EmailDomain: &emailDomainRule{Domain: rule.EmailDomain}}
	default:
		return accessRule{IP: &ipRule{IP: rule.IPRange}}
	}
}

func toPolicyBody(payload PolicyPayload) policyBody {
	return policyBody{
		Decision: payload.Decision,
		Exclude:  toAccessRules(payload.Exclude),
		Include:  toAccessRules(payload.Include),
		Name:     payload.Name,
		Require:  toAccessRules(payload.Require),
	}
}

func toApplicationBody(payload ApplicationPayload) applicationBody {
	policies := payload.Policies
	if policies == nil {
		policies = []string{}
	}
	return applicationBody{
		Domain:          payload.Domain,
		Name:            payload.Name,
		Policies:        policies,
		SessionDuration: payload.SessionDuration,
		Type:            payload.Type,
	}
}

func (c *Client) CreatePolicy(ctx context.Context, payload PolicyPayload) (string, error) {
	var created createdResource
	if err := c.api.do(ctx, http.MethodPost, c.accountPath(policiesPath), nil, toPolicyBody(payload), &created); err != nil {
		return "", fmt.Errorf("create access policy: %w", err)
	}
	return created.ID, nil
}

func (c *Client) UpdatePolicy(ctx context.Context, id string, payload PolicyPayload) error {
	if err := c.api.do(ctx, http.MethodPut, c.accountPath(policyPath, id), nil, toPolicyBody(payload), nil); err != nil {
		return fmt.Errorf("update access policy %q: %w", id, err)
	}
	return nil
}

func (c *Client) DeletePolicy(ctx context.Context, id string) error {
	if err := c.api.do(ctx, http.MethodDelete, c.accountPath(policyPath, id), nil, nil, nil); err != nil {
		return fmt.Errorf("delete access policy %q: %w", id, err)
	}
	return nil
}

func (c *Client) CreateApplication(ctx context.Context, payload ApplicationPayload) (string, error) {
	var created createdResource
	if err := c.api.do(ctx, http.MethodPost, c.accountPath(applicationsPath), nil, toApplicationBody(payload), &created); err != nil {
		return "", fmt.Errorf("create access application: %w", err)
	}
	return created.ID, nil
}

func (c *Client) UpdateApplication(ctx context.Context, id string, payload ApplicationPayload) error {
	if err := c.api.do(ctx, http.MethodPut, c.accountPath(applicationPath, id), nil, toApplicationBody(payload), nil); err != nil {
		return fmt.Errorf("update access application %q: %w", id, err)
	}
	return nil
}

func (c *Client) DeleteApplication(ctx context.Context, id string) error {
	if err := c.api.do(ctx, http.MethodDelete, c.accountPath(applicationPath, id), nil, nil, nil); err != nil {
		return fmt.Errorf("delete access application %q: %w", id, err)
	}
	return nil
}

type Route struct {
	ID      string
	Network string
}

type tunnelBody struct {
	ConfigSource string `json:"config_src"`
	Name         string `json:"name"`
}

type routeBody struct {
	Network  string `json:"network"`
	TunnelID string `json:"tunnel_id"`
}

type routeResult struct {
	ID      string `json:"id"`
	Network string `json:"network"`
}

func (c *Client) CreateTunnel(ctx context.Context, name string) (string, error) {
	var created createdResource
	body := tunnelBody{ConfigSource: tunnelConfigSourceCloudflare, Name: name}
	if err := c.api.do(ctx, http.MethodPost, c.accountPath(tunnelsPath), nil, body, &created); err != nil {
		return "", fmt.Errorf("create tunnel: %w", err)
	}
	return created.ID, nil
}

func (c *Client) DeleteTunnel(ctx context.Context, id string) error {
	if err := c.api.do(ctx, http.MethodDelete, c.accountPath(tunnelPath, id), nil, nil, nil); err != nil {
		return fmt.Errorf("delete tunnel %q: %w", id, err)
	}
	return nil
}

// TunnelToken is safe to call on every reconcile without caching: Cloudflare derives it
// deterministically from the tunnel's stored secret, so it's stable across calls.
func (c *Client) TunnelToken(ctx context.Context, id string) (string, error) {
	var token string
	if err := c.api.do(ctx, http.MethodGet, c.accountPath(tunnelTokenPath, id), nil, nil, &token); err != nil {
		return "", fmt.Errorf("fetch tunnel token %q: %w", id, err)
	}
	return token, nil
}

func (c *Client) ListRoutes(ctx context.Context, tunnelID string) ([]Route, error) {
	query := url.Values{tunnelIDQueryParameter: {tunnelID}}
	results, err := listAllPages[routeResult](ctx, c.api, c.accountPath(routesPath), query)
	if err != nil {
		return nil, fmt.Errorf("list routes for tunnel %q: %w", tunnelID, err)
	}
	var routes []Route
	for _, result := range results {
		routes = append(routes, Route(result))
	}
	return routes, nil
}

func (c *Client) CreateRoute(ctx context.Context, tunnelID, network string) error {
	body := routeBody{Network: network, TunnelID: tunnelID}
	if err := c.api.do(ctx, http.MethodPost, c.accountPath(routesPath), nil, body, nil); err != nil {
		return fmt.Errorf("create route %q: %w", network, err)
	}
	return nil
}

func (c *Client) DeleteRoute(ctx context.Context, routeID string) error {
	if err := c.api.do(ctx, http.MethodDelete, c.accountPath(routePath, routeID), nil, nil, nil); err != nil {
		return fmt.Errorf("delete route %q: %w", routeID, err)
	}
	return nil
}

type IngressRule struct {
	Hostname      string
	Service       string
	OriginRequest *OriginRequest
}

type OriginRequest struct {
	MatchSNIToHost bool
}

type ingressBody struct {
	Hostname      string             `json:"hostname,omitempty"`
	OriginRequest *originRequestBody `json:"originRequest,omitempty"`
	Service       string             `json:"service"`
}

type originRequestBody struct {
	MatchSNIToHost bool `json:"matchSNItoHost"`
}

type tunnelConfigurationBody struct {
	Config tunnelConfiguration `json:"config"`
}

type tunnelConfiguration struct {
	Ingress []ingressBody `json:"ingress"`
}

func (c *Client) PutTunnelConfiguration(ctx context.Context, tunnelID string, rules []IngressRule) error {
	ingress := make([]ingressBody, len(rules))
	for index, rule := range rules {
		ingress[index] = ingressBody{Hostname: rule.Hostname, Service: rule.Service}
		if rule.OriginRequest != nil {
			ingress[index].OriginRequest = &originRequestBody{MatchSNIToHost: rule.OriginRequest.MatchSNIToHost}
		}
	}

	body := tunnelConfigurationBody{Config: tunnelConfiguration{Ingress: ingress}}
	if err := c.api.do(ctx, http.MethodPut, c.accountPath(tunnelConfigurationPath, tunnelID), nil, body, nil); err != nil {
		return fmt.Errorf("update tunnel configuration %q: %w", tunnelID, err)
	}
	return nil
}
