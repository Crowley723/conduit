package handlers

import (
	"fmt"
	"github.com/Crowley723/conduit/internal/models"
	"github.com/Crowley723/conduit/internal/testutil"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
)

var (
	failedAuth              = "authentication failed"
	exampleErrorDescription = "This is the detailed description of the error!"
	exampleErrorUri         = "https://error.com/error_code/123456"
	exampleErrorState       = "abcdefg1234567"
	serverError             = "server error"
)

func TestGetCallbackHandler_ShouldRedirectOnSuccess(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "GET", "/api/auth/callback")
	tc.Request.URL.Query().Set("error", "")
	defer tc.Finish()

	testUser := &models.User{
		Sub:      "sub_claim",
		Iss:      "iss_claim",
		Username: "steve",
		Email:    "steve@example.com",
		Groups:   []string{"admin", "dev"},
	}
	idToken := &oidc.IDToken{}

	tc.MockOidcProvider.EXPECT().HandleCallback(tc.AppContext).Return(idToken, testUser, nil).Times(1)
	tc.MockStorageProvider.EXPECT().UpsertUser(tc.AppContext, testUser.Sub, testUser.Iss, testUser.Username, testUser.DisplayName, testUser.Email, testUser.Groups).Return(testUser, nil).Times(1)
	tc.MockSession.EXPECT().CreateSessionWithTokenExpiry(tc.AppContext, idToken, testUser)
	tc.MockSession.EXPECT().GetRedirectAfterLogin(tc.AppContext).Return("").Times(1)

	tc.CallHandler(GETCallbackHandler)

	tc.AssertStatus(t, http.StatusFound)
	tc.AssertContentType(t, "text/html; charset=utf-8")
	tc.AssertLogsContainMessage(t, slog.LevelDebug, "User successfully authenticated")
}

func TestGetCallbackHandler_ShouldErrorOnError(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "GET", "/api/auth/callback")
	tc.WithQueryParam("error", failedAuth)
	tc.WithQueryParam("error_description", exampleErrorDescription)
	tc.WithQueryParam("error_uri", exampleErrorUri)
	tc.WithQueryParam("state", exampleErrorState)
	defer tc.Finish()

	tc.CallHandler(GETCallbackHandler)

	expectedRedirect := fmt.Sprintf("/error?error=%s&error_description=%s&error_uri=%s&state=%s",
		url.QueryEscape(failedAuth),
		url.QueryEscape(exampleErrorDescription),
		url.QueryEscape(exampleErrorUri),
		url.QueryEscape(exampleErrorState))

	tc.AssertStatus(t, http.StatusFound)
	tc.AssertContentType(t, "text/html; charset=utf-8")
	tc.AssertLocationHeader(t, expectedRedirect)
	tc.AssertLogsContainMessage(t, slog.LevelWarn, "OIDC callback error")
}

func TestGetCallbackHandler_ShouldErrorOnCallbackError(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "GET", "/api/auth/callback")
	defer tc.Finish()

	tc.MockOidcProvider.EXPECT().HandleCallback(tc.AppContext).Return(nil, nil, fmt.Errorf("error")).Times(1)

	tc.CallHandler(GETCallbackHandler)

	expectedRedirect := fmt.Sprintf("/error?error=%s&error_description=%s",
		url.QueryEscape(serverError),
		url.QueryEscape(failedAuth))

	tc.AssertStatus(t, http.StatusFound)
	tc.AssertContentType(t, "text/html; charset=utf-8")
	tc.AssertLocationHeader(t, expectedRedirect)
	tc.AssertLogsContainMessage(t, slog.LevelError, "Failed to handle OIDC callback")
}

func TestGetCallbackHandler_ShouldRedirectToPreAuthLocation(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "GET", "/api/auth/callback")
	defer tc.Finish()

	testUser := &models.User{
		Sub:      "sub_claim",
		Iss:      "iss_claim",
		Username: "steve",
		Email:    "steve@example.com",
		Groups:   []string{"admin", "dev"},
	}
	idToken := &oidc.IDToken{}

	tc.MockOidcProvider.EXPECT().HandleCallback(tc.AppContext).Return(idToken, testUser, nil).Times(1)
	tc.MockStorageProvider.EXPECT().UpsertUser(tc.AppContext, testUser.Sub, testUser.Iss, testUser.Username, testUser.DisplayName, testUser.Email, testUser.Groups).Return(testUser, nil).Times(1)
	tc.MockSession.EXPECT().CreateSessionWithTokenExpiry(tc.AppContext, idToken, testUser)

	tc.MockSession.EXPECT().GetRedirectAfterLogin(tc.AppContext).Return(nonDefaultRedirectURL).Times(1)

	tc.CallHandler(GETCallbackHandler)

	tc.AssertStatus(t, http.StatusFound)
	tc.AssertContentType(t, "text/html; charset=utf-8")
	tc.AssertLocationHeader(t, nonDefaultRedirectURL)
}
