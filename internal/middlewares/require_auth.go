package middlewares

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/models"
	"github.com/Crowley723/conduit/internal/utils"
	"net/http"
	"strings"
	"time"

	"github.com/go-crypt/crypt"
	"github.com/go-crypt/crypt/algorithm/argon2"
)

var (
	serviceAccountTokenPrefix = "conduit_sa"
)

var (
	ErrInvalidServiceToken  = errors.New("invalid service token")
	ErrServiceTokenDisabled = errors.New("service account is disabled")
	ErrServiceTokenDeleted  = errors.New("service account has been deleted")
	ErrServiceTokenExpired  = errors.New("service token is expired")
)

func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appCtx := GetAppContext(r)
		if appCtx == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if appCtx.Config.Storage != nil {
			if authToken, err := utils.ExtractAuthorizationHeader(r); err == nil {
				if serviceAccount, err := VerifyServiceAccount(appCtx, authToken); err == nil && serviceAccount != nil {
					appCtx.SetPrincipal(serviceAccount)
					next.ServeHTTP(w, r)
					return
				}
				appCtx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
				return
			}
		}

		if user, ok := appCtx.SessionManager.GetUser(appCtx); ok {
			appCtx.SetPrincipal(user)
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appCtx := GetAppContext(r)
		if appCtx == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if appCtx.Config.Storage != nil {
			if authToken, err := utils.ExtractAuthorizationHeader(r); err == nil {
				if serviceAccount, err := VerifyServiceAccount(appCtx, authToken); err == nil && serviceAccount != nil {
					appCtx.SetPrincipal(serviceAccount)
					next.ServeHTTP(w, r)
					return
				}
				appCtx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
				return
			}
		}

		if user, ok := appCtx.SessionManager.GetUser(appCtx); ok {
			appCtx.SetPrincipal(user)
			next.ServeHTTP(w, r)
			return
		}

		principal := appCtx.GetPrincipal()
		if principal == nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequireCookieAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appCtx := GetAppContext(r)
		if appCtx == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		user, ok := appCtx.GetPrincipal().(*models.User)
		if !ok || user == nil {
			appCtx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireServiceAccountAuth requires a properly formatted and valid api bearer token.
// TODO: rate limiting
func RequireServiceAccountAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appCtx := GetAppContext(r)
		if appCtx == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		serviceAccount, ok := appCtx.GetPrincipal().(*models.ServiceAccount)
		if !ok || serviceAccount == nil {
			appCtx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// VerifyServiceAccount takes a service token, gets the associated service account, and validates that the service token is valid.
func VerifyServiceAccount(ctx *AppContext, token string) (*models.ServiceAccount, error) {
	lookupId, secret, err := ParseServiceToken(token)
	if err != nil {
		return nil, fmt.Errorf("unable to parse service token: %w", err)
	}

	serviceAccount, err := ctx.Storage.GetServiceAccountByLookupId(ctx, lookupId)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve service account from storage: %w", err)
	}

	//TODO: refactor to mitigate timing attacks
	valid, err := VerifyAPIToken(secret, serviceAccount.SecretHash)
	if err != nil {
		return nil, fmt.Errorf("unable to verify api token: %w", err)
	}

	if !valid {
		return nil, ErrInvalidServiceToken
	}

	if serviceAccount.DeletedAt != nil {
		return nil, ErrServiceTokenDeleted
	}

	if serviceAccount.IsDisabled {
		return nil, ErrServiceTokenDisabled
	}

	if time.Now().After(serviceAccount.TokenExpiresAt) {
		return nil, ErrServiceTokenExpired
	}

	return serviceAccount, nil
}

func ParseServiceToken(s string) (lookupId, secret string, err error) {
	parts := strings.Split(s, ".")

	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid token format: expected 3 parts, got %d", len(parts))
	}

	if parts[0] != serviceAccountTokenPrefix {
		return "", "", fmt.Errorf("invalid token prefix: expected %s, got %s", serviceAccountTokenPrefix, parts[0])
	}

	lookupId = parts[1]
	secret = parts[2]

	if len(lookupId) != 22 {
		return "", "", fmt.Errorf("invalid lookup ID length: expected 22 characters, got %d", len(lookupId))
	}

	if len(secret) != 43 {
		return "", "", fmt.Errorf("invalid secret length: expected 43 characters, got %d", len(secret))
	}

	if _, err := base64.RawURLEncoding.DecodeString(lookupId); err != nil {
		return "", "", fmt.Errorf("invalid lookup ID encoding: %v", err)
	}

	if _, err := base64.RawURLEncoding.DecodeString(secret); err != nil {
		return "", "", fmt.Errorf("invalid secret encoding: %v", err)
	}

	return lookupId, secret, nil
}

func GenerateAPIToken() (rawToken, lookupId, hashedSecret string, err error) {
	secretBytes := make([]byte, 32)
	lookupBytes := make([]byte, 16)

	rand.Read(secretBytes)
	rand.Read(lookupBytes)

	encodedSecret := base64.RawURLEncoding.EncodeToString(secretBytes)
	encodedLookup := base64.RawURLEncoding.EncodeToString(lookupBytes)

	rawToken = fmt.Sprintf(
		"%s.%s.%s",
		serviceAccountTokenPrefix,
		encodedLookup,
		encodedSecret,
	)

	hashedSecret, err = HashAPIToken(encodedSecret)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to hash token: %v", err)
	}

	return rawToken, encodedLookup, hashedSecret, nil
}

func HashAPIToken(token string) (string, error) {
	hasher, err := argon2.New(
		argon2.WithProfileRFC9106LowMemory(),
	)

	if err != nil {
		return "", fmt.Errorf("failed to create argon2 hasher: %v", err)
	}

	digest, err := hasher.Hash(token)
	if err != nil {
		return "", err
	}

	return digest.Encode(), nil
}

func VerifyAPIToken(token, hash string) (bool, error) {
	decoder := crypt.NewDecoder()
	err := argon2.RegisterDecoderArgon2id(decoder)
	if err != nil {
		return false, fmt.Errorf("failed to register argon2 decoder: %v", err)
	}

	digest, err := decoder.Decode(hash)
	if err != nil {
		return false, err
	}

	return digest.MatchAdvanced(token)
}
