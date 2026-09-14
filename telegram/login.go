package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

const Issuer = "https://oauth.telegram.org"

type LoginClient struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	HTTPClient   *http.Client
}

type Identity struct {
	ID            int64  `json:"id"`
	Phone         string `json:"phone_number"`
	PhoneVerified bool   `json:"phone_number_verified"`
	Subject       string `json:"sub"`
}

func (c LoginClient) AuthorizationURL(state, verifier string) string {
	v := url.Values{
		"client_id": {c.ClientID}, "redirect_uri": {c.RedirectURL},
		"response_type": {"code"}, "scope": {"openid profile phone telegram:bot_access"},
		"state": {state}, "nonce": {state},
		"code_challenge": {Hash(verifier)}, "code_challenge_method": {"S256"},
	}
	return Issuer + "/auth?" + v.Encode()
}

func (c LoginClient) Exchange(ctx context.Context, code, verifier, nonce string) (Identity, error) {
	var identity Identity
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client = &copyClient
	ctx = oidc.ClientContext(ctx, client)
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {c.RedirectURL}, "client_id": {c.ClientID}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, Issuer+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return identity, loginError("login_request")
	}
	req.SetBasicAuth(c.ClientID, c.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return identity, loginError("telegram_connection")
	}
	defer res.Body.Close()
	var tokens struct {
		IDToken string `json:"id_token"`
	}
	if res.StatusCode != http.StatusOK {
		var failure struct {
			Code string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&failure)
		switch failure.Code {
		case "invalid_client":
			return identity, loginError("telegram_client_settings")
		case "invalid_grant":
			return identity, loginError("telegram_code_rejected")
		default:
			return identity, loginError("telegram_token_response")
		}
	}
	if json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&tokens) != nil || tokens.IDToken == "" {
		return identity, loginError("telegram_token_response")
	}
	keys := oidc.NewRemoteKeySet(ctx, Issuer+"/.well-known/jwks.json")
	validator := oidc.NewVerifier(Issuer, keys, &oidc.Config{ClientID: c.ClientID, SupportedSigningAlgs: []string{"RS256"}})
	token, err := validator.Verify(ctx, tokens.IDToken)
	if err != nil {
		return identity, loginError("telegram_token_verification")
	}
	if token.Nonce != nonce {
		return identity, loginError("telegram_nonce")
	}
	if token.IssuedAt.After(time.Now().Add(time.Minute)) {
		return identity, loginError("telegram_token_time")
	}
	if err := token.Claims(&identity); err != nil {
		return identity, loginError("telegram_identity")
	}
	identity.Phone = strings.TrimPrefix(identity.Phone, "+")
	if identity.Subject == "" || identity.ID <= 0 {
		return Identity{}, loginError("telegram_identity")
	}
	if !identity.PhoneVerified || !regexp.MustCompile(`^[1-9][0-9]{6,14}$`).MatchString(identity.Phone) {
		return Identity{}, loginError("telegram_phone_required")
	}
	return identity, nil
}
