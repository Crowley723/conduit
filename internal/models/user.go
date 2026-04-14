package models

import (
	"github.com/Crowley723/conduit/internal/config"
	"slices"
	"time"
)

type User struct {
	Sub          string    `json:"sub"`
	Iss          string    `json:"iss"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email"`
	Groups       []string  `json:"groups"`
	IsSystem     bool      `json:"is_system"`
	LastLoggedIn time.Time `json:"last_logged_in"`
	CreatedAt    time.Time `json:"created_at"`
}

func (u *User) GetIss() string {
	return u.Iss
}

func (u *User) GetSub() string {
	return u.Sub
}

func (u *User) GetUsername() string {
	return u.Username
}

func (u *User) GetDisplayName() string {
	return u.DisplayName
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetGroups() []string {
	return u.Groups
}

func (u *User) GetScopes(cfg *config.Config) []string {
	var scopes []string
	for _, group := range u.Groups {
		scopes = append(scopes, mapGroupToScopes(cfg, group)...)
	}
	return dedupe(scopes)
}

func (u *User) HasScope(cfg *config.Config, scope string) bool {
	scopes := u.GetScopes(cfg)

	return slices.Contains(scopes, scope)
}

func (u *User) MatchesOwner(iss, sub string) bool {
	return u.Iss == iss && u.Sub == sub
}

// mapGroupToScopes maps groups to the associated scopes based on the config.
func mapGroupToScopes(cfg *config.Config, group string) []string {
	scopes := cfg.Authorization.GroupScopes[group]
	if scopes == nil {
		return []string{}
	}

	return scopes
}

func dedupe(slice []string) []string {
	keys := make(map[string]bool)
	var list []string

	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
