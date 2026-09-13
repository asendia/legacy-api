package telegram

import (
	"context"
	"encoding/json"
	"errors"
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
		return identity, errors.New("cannot create login request")
	}
	req.SetBasicAuth(c.ClientID, c.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return identity, errors.New("Telegram login request failed")
	}
	defer res.Body.Close()
	var tokens struct {
		IDToken string `json:"id_token"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&tokens) != nil {
		return identity, errors.New("Telegram login failed")
	}
	keys := oidc.NewRemoteKeySet(ctx, Issuer+"/.well-known/jwks.json")
	validator := oidc.NewVerifier(Issuer, keys, &oidc.Config{ClientID: c.ClientID, SupportedSigningAlgs: []string{"RS256"}})
	token, err := validator.Verify(ctx, tokens.IDToken)
	if err != nil || token.Nonce != nonce || token.IssuedAt.After(time.Now().Add(time.Minute)) {
		return identity, errors.New("invalid Telegram login token")
	}
	if err := token.Claims(&identity); err != nil {
		return identity, errors.New("invalid Telegram identity")
	}
	identity.Phone = strings.TrimPrefix(identity.Phone, "+")
	if identity.Subject == "" || identity.ID <= 0 || !identity.PhoneVerified || !regexp.MustCompile(`^[1-9][0-9]{6,14}$`).MatchString(identity.Phone) {
		return Identity{}, errors.New("share your verified phone number to use Telegram login")
	}
	return identity, nil
}
