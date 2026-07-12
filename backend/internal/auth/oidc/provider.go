package oidc

import (
	"context"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	provider, err := gooidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}
	return &Provider{
		OAuth:    oauthConfig,
		Verifier: realTokenVerifier{verifier: provider.Verifier(&gooidc.Config{ClientID: cfg.ClientID})},
		UserInfo: realUserInfoFetcher{provider: provider},
	}, nil
}

func Nonce(nonce string) oauth2.AuthCodeOption {
	return gooidc.Nonce(nonce)
}

type realTokenVerifier struct {
	verifier *gooidc.IDTokenVerifier
}

func (v realTokenVerifier) Verify(ctx context.Context, rawIDToken string) (IDToken, error) {
	token, err := v.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return IDToken{}, err
	}
	return IDToken{
		Subject: token.Subject,
		Nonce:   token.Nonce,
	}, nil
}

type realUserInfoFetcher struct {
	provider *gooidc.Provider
}

func (f realUserInfoFetcher) Fetch(ctx context.Context, tokenSource oauth2.TokenSource) (Profile, error) {
	info, err := f.provider.UserInfo(ctx, tokenSource)
	if err != nil {
		return Profile{}, err
	}
	var claims struct {
		Subject           string `json:"sub"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := info.Claims(&claims); err != nil {
		return Profile{}, err
	}
	displayName := claims.Name
	if displayName == "" {
		displayName = claims.PreferredUsername
	}
	return Profile{
		Subject:     claims.Subject,
		Email:       claims.Email,
		DisplayName: displayName,
	}, nil
}
