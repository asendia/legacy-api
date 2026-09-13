package telegram

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLoginTokenChecks(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"valid", "wrong-audience", "wrong-issuer", "wrong-nonce", "expired", "unsigned", "phone-not-verified", "missing-phone", "future-issued"} {
		t.Run(name, func(t *testing.T) {
			claims := map[string]interface{}{"iss": Issuer, "aud": "123", "sub": "subject", "id": 12345, "phone_number": "+628123456789", "phone_number_verified": true, "nonce": "nonce", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()}
			switch name {
			case "wrong-audience":
				claims["aud"] = "other"
			case "wrong-issuer":
				claims["iss"] = "https://example.org"
			case "wrong-nonce":
				claims["nonce"] = "other"
			case "expired":
				claims["exp"] = time.Now().Add(-time.Minute).Unix()
			case "phone-not-verified":
				claims["phone_number_verified"] = false
			case "missing-phone":
				delete(claims, "phone_number")
			case "future-issued":
				claims["iat"] = time.Now().Add(time.Hour).Unix()
			}
			header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"test"}`))
			body, _ := json.Marshal(claims)
			unsigned := header + "." + base64.RawURLEncoding.EncodeToString(body)
			hash := sha256.Sum256([]byte(unsigned))
			signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
			if err != nil {
				t.Fatal(err)
			}
			token := unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
			if name == "unsigned" {
				token = unsigned + "."
			}
			client := LoginClient{ClientID: "123", ClientSecret: "secret", RedirectURL: "https://sejiwo.com/telegram/callback", HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				var result interface{}
				if r.URL.Path == "/token" {
					user, password, ok := r.BasicAuth()
					if !ok || user != "123" || password != "secret" {
						t.Fatal("missing client credentials")
					}
					if err := r.ParseForm(); err != nil {
						t.Fatal(err)
					}
					if r.Form.Get("code_verifier") != "verifier" || r.Form.Get("code") != "code" {
						t.Fatal("missing PKCE verifier")
					}
					result = map[string]string{"id_token": token}
				} else if r.URL.Path == "/.well-known/jwks.json" {
					result = map[string]interface{}{"keys": []interface{}{map[string]string{"kty": "RSA", "kid": "test", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()), "e": "AQAB"}}}
				} else {
					t.Fatalf("unexpected request %s", r.URL.Path)
				}
				encoded, _ := json.Marshal(result)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(encoded)))}, nil
			})}}
			identity, err := client.Exchange(context.Background(), "code", "verifier", "nonce")
			if name == "valid" {
				if err != nil || identity.Subject != "subject" || identity.Phone != "628123456789" {
					t.Fatalf("valid login failed: %v", err)
				}
			} else if err == nil {
				t.Fatal("invalid login accepted")
			}
		})
	}
}

func TestStoredContentTamper(t *testing.T) {
	key := "01234567890123456789012345678901"
	sealed, err := Seal("private", key)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Open(sealed, key)
	if err != nil || plain != "private" {
		t.Fatal("round trip failed")
	}
	b, err := base64.RawURLEncoding.DecodeString(sealed)
	if err != nil {
		t.Fatal(err)
	}
	b[len(b)-1] ^= 1
	if _, err := Open(base64.RawURLEncoding.EncodeToString(b), key); err == nil {
		t.Fatal("tampered content accepted")
	}
}
