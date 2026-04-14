package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/Crowley723/conduit/internal/authorization"
	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/models"
	"github.com/Crowley723/conduit/internal/storage"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type CertificateRequestResponse struct {
	ID                  int                             `json:"id"`
	OwnerIss            string                          `json:"owner_iss"`
	OwnerSub            string                          `json:"owner_sub"`
	OwnerUsername       string                          `json:"owner_username"`
	OwnerDisplayName    string                          `json:"owner_display_name"`
	Message             string                          `json:"message,omitempty"`
	Events              []models.CertificateEvent       `json:"events,omitempty"`
	CommonName          string                          `json:"common_name"`
	DNSNames            []string                        `json:"dns_names,omitempty"`
	OrganizationalUnits []string                        `json:"organizational_units,omitempty"`
	ValidityDays        int                             `json:"validity_days"`
	Status              models.CertificateRequestStatus `json:"status,omitempty"`
	RequestedAt         time.Time                       `json:"requested_at"`
	IssuedAt            *time.Time                      `json:"issued_at,omitempty"`
	ExpiresAt           *time.Time                      `json:"expires_at,omitempty"`
	SerialNumber        *string                         `json:"serial_number,omitempty"`
}

// POSTCertificateRequest is used by any authenticated user to create a certificate request
func POSTCertificateRequest(ctx *middlewares.AppContext) {
	principal := ctx.GetPrincipal()
	if principal == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	if !principal.HasScope(ctx.Config, authorization.ScopeMTLSRequestCert) {
		ctx.SetJSONError(http.StatusForbidden, http.StatusText(http.StatusForbidden))
		return
	}

	var req struct {
		Message      string `json:"message"`
		ValidityDays int    `json:"validity_days"`
	}

	if err := json.NewDecoder(ctx.Request.Body).Decode(&req); err != nil {
		ctx.Logger.Error("failed to decode request body", "error", err)
		ctx.SetJSONError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	if req.ValidityDays == 0 {
		req.ValidityDays = 90
	}

	if req.ValidityDays < ctx.Config.Features.MTLSManagement.MinCertificateValidityDays || req.ValidityDays > ctx.Config.Features.MTLSManagement.MaxCertificateValidityDays {
		ctx.SetJSONError(http.StatusBadRequest, fmt.Sprintf("validity_days must be between %d and %d days", ctx.Config.Features.MTLSManagement.MinCertificateValidityDays, ctx.Config.Features.MTLSManagement.MaxCertificateValidityDays))
		return
	}

	commonName := deriveCommonName(principal)

	requestStatus := string(models.StatusAwaitingReview)

	certRequest, err := ctx.Storage.CreateCertificateRequest(
		ctx,
		principal.GetSub(),
		principal.GetIss(),
		commonName,
		string(models.StatusAwaitingReview),
		req.Message,
		[]string{}, //empty dns name for client certs
		nil,
		req.ValidityDays,
	)

	if err != nil {
		ctx.Logger.Error("failed to create certificate request",
			"error", err,
			"principal_name", principal.GetUsername(),
			"common_name", commonName,
		)

		ctx.SetJSONError(http.StatusInternalServerError, "failed to create certificate request")
		return
	}

	ctx.Logger.Debug("certificate request created",
		"request_id", certRequest.ID,
		"principal_name", principal.GetUsername(),
		"common_name", commonName,
	)

	if principal.HasScope(ctx.Config, authorization.ScopeMTLSAutoApproveCert) {
		err = ctx.Storage.UpdateCertificateRequestStatus(ctx, certRequest.ID, models.StatusApproved, ctx.Config.Server.ExternalURL, storage.SystemSub, "Auto Approved")
		if err != nil {
			ctx.Logger.Error("failed to auto approve certificate request", "error", err)
			ctx.SetJSONError(http.StatusInternalServerError, "Failed to auto approve certificate request")
			return
		}
		ctx.Logger.Debug("request was auto-approved", "iss", principal.GetIss(), "sub", principal.GetSub(), "request_status", requestStatus)
	}

	updatedRequest, err := ctx.Storage.GetCertificateRequestByID(ctx, certRequest.ID)
	if err != nil {
		ctx.Logger.Error("failed to get certificate requests",
			"error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Failed to get certificate requests")
		return
	}

	ctx.WriteJSON(http.StatusCreated, updatedRequest)
}

// GETCertificateRequests is used to expose all certificate requests to admin users. Admin check done with middleware
func GETCertificateRequests(ctx *middlewares.AppContext) {
	principal := ctx.GetPrincipal()
	if principal == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	if !principal.HasScope(ctx.Config, authorization.ScopeMTLSReadAllCerts) {
		ctx.SetJSONError(http.StatusForbidden, http.StatusText(http.StatusForbidden))
		return
	}

	requests, err := ctx.Storage.GetCertificateRequests(ctx)
	if err != nil {
		ctx.Logger.Error("failed to get certificate requests",
			"error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Failed to get certificate requests")
		return
	}

	if requests == nil {
		ctx.WriteJSON(http.StatusOK, []interface{}{})
		return
	}

	ctx.WriteJSON(http.StatusOK, redactCertificateFields(requests))
}

// GETCertificateRequest is used to expose a single certificate request
func GETCertificateRequest(ctx *middlewares.AppContext) {
	requestIdParam := chi.URLParam(ctx.Request, "id")
	if requestIdParam == "" {
		ctx.SetJSONError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	requestId, err := strconv.Atoi(strings.TrimSpace(requestIdParam))
	if err != nil {
		ctx.SetJSONError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	principal := ctx.GetPrincipal()
	if principal == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	requests, err := ctx.Storage.GetCertificateRequestByID(ctx, requestId)
	if err != nil {
		ctx.Logger.Error("failed to get certificate requests",
			"error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Failed to get certificate requests")
		return
	}

	if requests != nil &&
		!principal.MatchesOwner(requests.OwnerIss, requests.OwnerSub) &&
		!principal.HasScope(ctx.Config, authorization.ScopeMTLSReadAllCerts) {
		ctx.SetJSONError(http.StatusForbidden, http.StatusText(http.StatusForbidden))
		return
	}

	if requests == nil {
		ctx.WriteJSON(http.StatusOK, []interface{}{})
		return
	}

	ctx.WriteJSON(http.StatusOK, redactCertificateFields(
		[]*models.CertificateRequest{
			requests,
		}))
}

// POSTCertificateReview is used by admins to post a reject/approval for a specific certificate request with comments.
func POSTCertificateReview(ctx *middlewares.AppContext) {
	requestIdParam := chi.URLParam(ctx.Request, "id")
	if requestIdParam == "" {
		ctx.SetJSONError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	requestId, err := strconv.Atoi(strings.TrimSpace(requestIdParam))
	if err != nil {
		ctx.SetJSONError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	principal := ctx.GetPrincipal()
	if principal == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	if !principal.HasScope(ctx.Config, authorization.ScopeMTLSApproveCert) {
		ctx.SetJSONError(http.StatusForbidden, http.StatusText(http.StatusForbidden))
		return
	}

	var review struct {
		NewStatus   models.CertificateRequestStatus `json:"new_status"`
		ReviewNotes string                          `json:"review_notes"`
	}

	if err := json.NewDecoder(ctx.Request.Body).Decode(&review); err != nil {
		ctx.SetJSONError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	review.NewStatus = models.CertificateRequestStatus(strings.TrimSpace(string(review.NewStatus)))

	if review.NewStatus != models.StatusApproved && review.NewStatus != models.StatusRejected {
		ctx.SetJSONError(http.StatusBadRequest,
			"Invalid status. Must be 'approved' or 'rejected'")
		return
	}

	request, err := ctx.Storage.GetCertificateRequestByID(ctx, requestId)
	if err != nil {
		ctx.Logger.Error("failed to get certificate requests",
			"error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Failed to fetch certificate request")
		return
	}

	if request == nil {
		ctx.SetJSONStatus(http.StatusOK, "No certificate requests found")
		return
	}

	if !principal.HasScope(ctx.Config, authorization.ScopeMTLSSelfApproveCerts) && principal.MatchesOwner(request.OwnerIss, request.OwnerSub) {
		ctx.SetJSONError(http.StatusForbidden, "You are not allowed to approve your own requests")
		return
	}

	if request.Status != models.StatusAwaitingReview {
		ctx.SetJSONError(http.StatusBadRequest,
			fmt.Sprintf("Cannot review request with status '%s'. Only requests with status 'awaiting_review' can be reviewed.",
				request.Status))
		return
	}

	err = ctx.Storage.UpdateCertificateRequestStatus(ctx, request.ID, review.NewStatus, principal.GetIss(), principal.GetSub(), review.ReviewNotes)
	if err != nil {
		ctx.Logger.Error("failed to update certificate request status", "error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Failed to update certificate request status")
		return
	}

	ctx.Logger.Info("certificate request reviewed",
		"request_id", requestId,
		"reviewer", principal.GetUsername(),
		"new_status", review.NewStatus,
	)

	updatedRequest, err := ctx.Storage.GetCertificateRequestByID(ctx, requestId)
	if err != nil {
		ctx.Logger.Error("failed to get certificate requests",
			"error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Failed to get certificate requests")
		return
	}

	ctx.WriteJSON(http.StatusOK, updatedRequest)
}

// GETUserCertificateRequests exposes information about certificate requests to the owner
func GETUserCertificateRequests(ctx *middlewares.AppContext) {
	principal := ctx.GetPrincipal()
	if principal == nil {
		ctx.SetJSONError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return
	}

	requests, err := ctx.Storage.GetCertificateRequestsByUser(ctx, principal.GetSub(), principal.GetIss())
	if err != nil {
		ctx.Logger.Error("failed to get certificate requests",
			"error", err)
		ctx.SetJSONError(http.StatusInternalServerError, "Failed to get certificate requests")
		return
	}

	if requests == nil {
		ctx.WriteJSON(http.StatusOK, []interface{}{})
		return
	}

	ctx.WriteJSON(http.StatusOK, redactCertificateFields(requests))
}

func redactCertificateFields(requests []*models.CertificateRequest) []*models.CertificateRequest {
	result := make([]*models.CertificateRequest, len(requests))
	for i, req := range requests {
		reqCopy := *req
		reqCopy.CertificateIdentifier = nil
		reqCopy.ProviderMetadata = nil
		reqCopy.CertificatePem = nil
		result[i] = &reqCopy
	}
	return result
}
