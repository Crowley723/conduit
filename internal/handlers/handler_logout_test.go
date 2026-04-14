package handlers

import (
	"errors"
	"github.com/Crowley723/conduit/internal/models"
	"github.com/Crowley723/conduit/internal/testutil"
	"log/slog"
	"net/http"
	"testing"
)

func TestLogoutHandler_ShouldDestroySession(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "POST", "/api/auth/logout")
	defer tc.Finish()

	tc.MockSession.EXPECT().IsUserAuthenticated(tc.AppContext).Return(true)

	testUser := &models.User{
		Sub:      "sub_claim",
		Iss:      "iss_claim",
		Username: "steve",
	}

	tc.MockSession.EXPECT().GetUser(tc.AppContext).Return(testUser, true)

	tc.MockSession.EXPECT().Logout(tc.AppContext).Return(nil)

	tc.CallHandler(POSTLogoutHandler)

	tc.AssertStatus(t, http.StatusOK)
	tc.AssertContentType(t, "application/json")
	tc.AssertJSONField(t, "status", "OK")
}

func TestLogoutHandler_Should401AnonymousUsers(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "POST", "/api/auth/logout")
	defer tc.Finish()

	tc.MockSession.EXPECT().IsUserAuthenticated(tc.AppContext).Return(false)

	tc.CallHandler(POSTLogoutHandler)

	tc.AssertStatus(t, http.StatusBadRequest)
	tc.AssertContentType(t, "application/json")
	tc.AssertJSONField(t, "error", "Bad Request")
}

func TestLogoutHandler_Should500OnUnknownUser(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "POST", "/api/auth/logout")
	defer tc.Finish()

	tc.MockSession.EXPECT().IsUserAuthenticated(tc.AppContext).Return(true)

	tc.MockSession.EXPECT().GetUser(tc.AppContext).Return(nil, true)

	tc.CallHandler(POSTLogoutHandler)

	tc.AssertStatus(t, http.StatusInternalServerError)
	tc.AssertContentType(t, "application/json")
	tc.AssertJSONField(t, "error", "Internal Server Error")
}

func TestLogoutHandler_Should500OnLogoutFail(t *testing.T) {
	tc := testutil.NewTestContextWithURL(t, "POST", "/api/auth/logout")
	defer tc.Finish()

	tc.MockSession.EXPECT().IsUserAuthenticated(tc.AppContext).Return(true)

	testUser := &models.User{
		Sub:      "sub_claim",
		Iss:      "iss_claim",
		Username: "steve",
	}

	tc.MockSession.EXPECT().GetUser(tc.AppContext).Return(testUser, true)

	tc.MockSession.EXPECT().Logout(tc.AppContext).Return(errors.New("fail"))

	tc.CallHandler(POSTLogoutHandler)

	tc.AssertStatus(t, http.StatusInternalServerError)
	tc.AssertContentType(t, "application/json")
	tc.AssertJSONField(t, "error", "Internal Server Error")
	tc.AssertLogsContainMessage(t, slog.LevelError, "Failed to logout user")
}
