// Package auth applies a request's authentication config to an outgoing
// *http.Request. Header/query schemes mutate the request in place; OAuth2
// performs a token-fetch round-trip first and sends the result as a Bearer
// token. All credential values support {{var}} interpolation via the scope.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"senda/internal/model"
	"senda/internal/vars"
)

// Apply attaches the configured auth to r. hc is used only by the OAuth2 flow
// to fetch a token. A zero/none/inherit Auth is a no-op (inherit is resolved
// upstream before this is called).
func Apply(ctx context.Context, hc *http.Client, r *http.Request, a model.Auth, scope *vars.Scope) error {
	switch a.Type {
	case model.AuthBearer:
		r.Header.Set("Authorization", "Bearer "+scope.Apply(a.Token))
	case model.AuthBasic:
		r.SetBasicAuth(scope.Apply(a.Username), scope.Apply(a.Password))
	case model.AuthAPIKey:
		key := scope.Apply(a.Key)
		if key == "" {
			break
		}
		val := scope.Apply(a.KeyValue)
		if a.Placement == model.APIKeyQuery {
			q := r.URL.Query()
			q.Set(key, val)
			r.URL.RawQuery = q.Encode()
		} else {
			r.Header.Set(key, val)
		}
	case model.AuthOAuth2:
		tok, err := fetchOAuth2Token(ctx, hc, a, scope)
		if err != nil {
			return err
		}
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	return nil
}

// fetchOAuth2Token requests an access token from the token endpoint using a
// non-interactive grant (client_credentials by default, or password).
func fetchOAuth2Token(ctx context.Context, hc *http.Client, a model.Auth, scope *vars.Scope) (string, error) {
	tokenURL := scope.Apply(a.TokenURL)
	if tokenURL == "" {
		return "", fmt.Errorf("oauth2: token URL is empty")
	}

	grant := a.Grant
	if grant == "" {
		grant = model.OAuth2ClientCredentials
	}

	form := url.Values{}
	form.Set("grant_type", string(grant))
	form.Set("client_id", scope.Apply(a.ClientID))
	form.Set("client_secret", scope.Apply(a.ClientSecret))
	if s := scope.Apply(a.Scope); s != "" {
		form.Set("scope", s)
	}
	if grant == model.OAuth2Password {
		form.Set("username", scope.Apply(a.OAuthUsername))
		form.Set("password", scope.Apply(a.OAuthPassword))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth2: token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("oauth2: token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("oauth2: malformed token response: %w", err)
	}
	if tr.AccessToken == "" {
		if tr.Error != "" {
			return "", fmt.Errorf("oauth2: %s %s", tr.Error, tr.ErrorDesc)
		}
		return "", fmt.Errorf("oauth2: response had no access_token")
	}
	return tr.AccessToken, nil
}
