package middlewares

import (
	"github.com/Crowley723/conduit/internal/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

//go:generate mockgen -source=oidc_provider.go -destination=../mocks/oidc.go -package=mocks

type OIDCProvider interface {
	GenerateRandString(bytes int) string
	StartLogin(ctx *AppContext) (string, error)
	HandleCallback(ctx *AppContext) (*oidc.IDToken, *models.User, error)
	GetProvider() *oidc.Provider
	GetOAuth2Config() *oauth2.Config
}
