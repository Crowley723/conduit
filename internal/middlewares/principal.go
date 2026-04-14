package middlewares

import (
	"github.com/Crowley723/conduit/internal/config"
)

//go:generate mockgen -source=principal.go -destination=../mocks/principal.go -package=mocks

type Principal interface {
	GetIss() string
	GetSub() string
	GetUsername() string
	GetDisplayName() string
	GetEmail() string
	GetGroups() []string
	GetScopes(cfg *config.Config) []string
	HasScope(cfg *config.Config, scope string) bool
	MatchesOwner(iss, sub string) bool
}
