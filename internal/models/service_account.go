package models

import (
	"github.com/Crowley723/conduit/internal/config"
	"slices"
	"time"
)

type ServiceAccount struct {
	Sub            string     `json:"sub"`
	Iss            string     `json:"iss"`
	Name           string     `json:"name"`
	LookupId       string     `json:"lookup_id"`
	SecretHash     string     `json:"secret_hash"`
	TokenExpiresAt time.Time  `json:"token_expires_at"`
	Scopes         []string   `json:"scopes"`
	IsDisabled     bool       `json:"is_disabled"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	CreatedBySub   string     `json:"created_by_sub"`
	CreatedByIss   string     `json:"created_by_iss"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (s ServiceAccount) GetIss() string {
	return s.Iss
}

func (s ServiceAccount) GetSub() string {
	return s.Sub
}

func (s ServiceAccount) GetUsername() string {
	return s.Name
}

func (s ServiceAccount) GetDisplayName() string {
	return s.Name
}

func (s ServiceAccount) GetEmail() string {
	return ""
}

func (s ServiceAccount) GetGroups() []string {
	return []string{}
}

func (s ServiceAccount) GetScopes(cfg *config.Config) []string {
	return s.Scopes
}

func (s ServiceAccount) HasScope(cfg *config.Config, scope string) bool {
	return slices.Contains(s.GetScopes(cfg), scope)
}

func (s ServiceAccount) MatchesOwner(iss, sub string) bool {
	return s.Iss == iss && s.Sub == sub
}
