package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateCertificateRequest adds a certificate request to the database.
func (p *DatabaseProvider) CreateCertificateRequest(ctx context.Context, sub, iss, commonName, status, message string, dnsNames, organizationalUnits []string, validityDays int) (*models.CertificateRequest, error) {
	query := `
		INSERT INTO certificate_requests (owner_sub, owner_iss, common_name, status, message, dns_names, organizational_units, validity_days)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var requestID int
	err := p.pool.QueryRow(ctx, query,
		sub, iss, commonName, status, message,
		dnsNames, organizationalUnits, validityDays,
	).Scan(&requestID)

	if err != nil {
		return nil, fmt.Errorf("failed to create certificate request: %w", err)
	}

	return p.GetCertificateRequestByID(ctx, requestID)
}

func (p *DatabaseProvider) GetCertificateRequestByID(ctx context.Context, id int) (*models.CertificateRequest, error) {
	query := `
		SELECT id, owner_iss, owner_sub, message, common_name, dns_names, organizational_units, validity_days, status, requested_at, certificate_identifier, provider_metadata, issued_at, expires_at, serial_number, certificate_pem
		FROM certificate_requests
		WHERE id = $1
	`

	eventsQuery := `
		SELECT id, certificate_request_id, requester_iss, requester_sub, reviewer_iss, reviewer_sub, new_status, review_notes, created_at
		FROM certificate_events
		WHERE certificate_request_id = $1
		`

	var certificateRequest models.CertificateRequest
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&certificateRequest.ID,
		&certificateRequest.OwnerIss,
		&certificateRequest.OwnerSub,
		&certificateRequest.Message,
		&certificateRequest.CommonName,
		&certificateRequest.DNSNames,
		&certificateRequest.OrganizationalUnits,
		&certificateRequest.ValidityDays,
		&certificateRequest.Status,
		&certificateRequest.RequestedAt,
		&certificateRequest.CertificateIdentifier,
		&certificateRequest.ProviderMetadata,
		&certificateRequest.IssuedAt,
		&certificateRequest.ExpiresAt,
		&certificateRequest.SerialNumber,
		&certificateRequest.CertificatePem,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, CertificateRequestNotFoundError
		}
		return nil, fmt.Errorf("failed to get certificate request by id: %w", err)
	}

	rows, err := p.pool.Query(ctx, eventsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for certificate request '%d': %w", id, err)
	}
	defer rows.Close()

	var events []models.CertificateEvent
	for rows.Next() {
		var event models.CertificateEvent
		if err := rows.Scan(
			&event.ID,
			&event.CertificateRequestID,
			&event.RequesterIss,
			&event.RequesterSub,
			&event.ReviewerIss,
			&event.ReviewerSub,
			&event.NewStatus,
			&event.ReviewNotes,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate certificate events: %w", err)
	}

	certificateRequest.Events = events
	if certificateRequest.Events == nil {
		certificateRequest.Events = []models.CertificateEvent{}
	}

	return &certificateRequest, nil
}

func (p *DatabaseProvider) GetCertificateRequests(ctx context.Context) ([]*models.CertificateRequest, error) {
	query := `
       SELECT id, owner_iss, owner_sub, message, common_name, dns_names, organizational_units, validity_days, status, requested_at, certificate_identifier, provider_metadata, issued_at, expires_at, serial_number, certificate_pem
       FROM certificate_requests
       ORDER BY requested_at DESC
    `

	eventsQuery := `
       SELECT id, certificate_request_id, requester_iss, requester_sub, reviewer_iss, reviewer_sub, new_status, review_notes, created_at
       FROM certificate_events
       WHERE certificate_request_id = ANY($1)
       ORDER BY certificate_request_id, created_at
    `

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.CertificateRequest
	var requestIDs []int

	for rows.Next() {
		var req models.CertificateRequest
		if err := rows.Scan(
			&req.ID,
			&req.OwnerIss,
			&req.OwnerSub,
			&req.Message,
			&req.CommonName,
			&req.DNSNames,
			&req.OrganizationalUnits,
			&req.ValidityDays,
			&req.Status,
			&req.RequestedAt,
			&req.CertificateIdentifier,
			&req.ProviderMetadata,
			&req.IssuedAt,
			&req.ExpiresAt,
			&req.SerialNumber,
			&req.CertificatePem,
		); err != nil {
			return nil, fmt.Errorf("failed to scan certificate request: %w", err)
		}
		req.Events = []models.CertificateEvent{}
		requests = append(requests, &req)
		requestIDs = append(requestIDs, req.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate certificate requests: %w", err)
	}

	if len(requests) == 0 {
		return requests, nil
	}

	// Fetch all events for all requests in one query
	eventRows, err := p.pool.Query(ctx, eventsQuery, requestIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for certificate requests: %w", err)
	}
	defer eventRows.Close()

	// CreateUser a map for quick lookup
	requestMap := make(map[int]*models.CertificateRequest)
	for _, req := range requests {
		requestMap[req.ID] = req
	}

	for eventRows.Next() {
		var event models.CertificateEvent
		if err := eventRows.Scan(
			&event.ID,
			&event.CertificateRequestID,
			&event.RequesterIss,
			&event.RequesterSub,
			&event.ReviewerIss,
			&event.ReviewerSub,
			&event.NewStatus,
			&event.ReviewNotes,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if req, ok := requestMap[event.CertificateRequestID]; ok {
			req.Events = append(req.Events, event)
		}
	}

	if err := eventRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate certificate events: %w", err)
	}

	return requests, nil
}

func (p *DatabaseProvider) GetCertificateRequestsByUser(ctx context.Context, sub, iss string) ([]*models.CertificateRequest, error) {
	query := `
       SELECT
			cr.id, cr.owner_iss, cr.owner_sub, cr.message, cr.common_name,
			cr.dns_names, cr.organizational_units, cr.validity_days, cr.status,
			cr.requested_at, cr.certificate_identifier, cr.provider_metadata,
			cr.issued_at, cr.expires_at, cr.serial_number, cr.certificate_pem,
			owner.username as owner_username,
			owner.display_name as owner_display_name
		FROM certificate_requests cr
		JOIN users owner ON cr.owner_iss = owner.iss AND cr.owner_sub = owner.sub
		WHERE cr.owner_sub = $1 AND cr.owner_iss = $2
		ORDER BY cr.requested_at DESC
    `

	eventsQuery := `
       SELECT 
			ce.id, ce.certificate_request_id, 
			ce.requester_iss, ce.requester_sub,
			requester.username as requester_username,
			requester.display_name as requester_display_name,
			ce.reviewer_iss, ce.reviewer_sub,
			reviewer.username as reviewer_username,
			reviewer.display_name as reviewer_display_name,
			ce.new_status, ce.review_notes, ce.created_at
		FROM certificate_events ce
		JOIN users requester ON ce.requester_iss = requester.iss AND ce.requester_sub = requester.sub
		JOIN users reviewer ON ce.reviewer_iss = reviewer.iss AND ce.reviewer_sub = reviewer.sub
		WHERE ce.certificate_request_id = ANY($1)
		ORDER BY ce.certificate_request_id, ce.created_at
    `

	rows, err := p.pool.Query(ctx, query, sub, iss)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate requests for user '%s/%s': %w", iss, sub, err)
	}
	defer rows.Close()

	var requests []*models.CertificateRequest
	var requestIDs []int

	for rows.Next() {
		var req models.CertificateRequest
		if err := rows.Scan(
			&req.ID,
			&req.OwnerIss,
			&req.OwnerSub,
			&req.Message,
			&req.CommonName,
			&req.DNSNames,
			&req.OrganizationalUnits,
			&req.ValidityDays,
			&req.Status,
			&req.RequestedAt,
			&req.CertificateIdentifier,
			&req.ProviderMetadata,
			&req.IssuedAt,
			&req.ExpiresAt,
			&req.SerialNumber,
			&req.CertificatePem,
			&req.OwnerUsername,
			&req.OwnerDisplayName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan certificate request: %w", err)
		}
		req.Events = []models.CertificateEvent{}
		requests = append(requests, &req)
		requestIDs = append(requestIDs, req.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate certificate requests: %w", err)
	}

	if len(requests) == 0 {
		return requests, nil
	}

	// Fetch all events for all requests in one query
	eventRows, err := p.pool.Query(ctx, eventsQuery, requestIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for certificate requests: %w", err)
	}
	defer eventRows.Close()

	// CreateUser a map for quick lookup
	requestMap := make(map[int]*models.CertificateRequest)
	for _, req := range requests {
		requestMap[req.ID] = req
	}

	for eventRows.Next() {
		var event models.CertificateEvent
		if err := eventRows.Scan(
			&event.ID,
			&event.CertificateRequestID,
			&event.RequesterIss,
			&event.RequesterSub,
			&event.RequesterUsername,
			&event.RequesterDisplayName,
			&event.ReviewerIss,
			&event.ReviewerSub,
			&event.ReviewerUsername,
			&event.ReviewerDisplayName,
			&event.NewStatus,
			&event.ReviewNotes,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if req, ok := requestMap[event.CertificateRequestID]; ok {
			req.Events = append(req.Events, event)
		}
	}

	if err := eventRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate certificate events: %w", err)
	}

	return requests, nil
}

func (p *DatabaseProvider) GetCertificateRequestsPaginated(ctx context.Context, params models.PaginationParams) (*models.PaginatedCertResult, error) {
	// Set default limit if not provided
	if params.Limit <= 0 {
		params.Limit = 20
	}
	// Cap maximum limit to prevent abuse
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM certificate_requests`
	var total int
	if err := p.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get paginated requests
	query := `
       SELECT id, owner_iss, owner_sub, message, common_name, dns_names, organizational_units, validity_days, status, requested_at, certificate_identifier, provider_metadata, issued_at, expires_at, serial_number, certificate_pem
       FROM certificate_requests
       ORDER BY requested_at DESC
       LIMIT $1 OFFSET $2
    `

	rows, err := p.pool.Query(ctx, query, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.CertificateRequest
	var requestIDs []int

	for rows.Next() {
		var req models.CertificateRequest
		if err := rows.Scan(
			&req.ID,
			&req.OwnerIss,
			&req.OwnerSub,
			&req.Message,
			&req.CommonName,
			&req.DNSNames,
			&req.OrganizationalUnits,
			&req.ValidityDays,
			&req.Status,
			&req.RequestedAt,
			&req.CertificateIdentifier,
			&req.ProviderMetadata,
			&req.IssuedAt,
			&req.ExpiresAt,
			&req.SerialNumber,
			&req.CertificatePem,
		); err != nil {
			return nil, fmt.Errorf("failed to scan certificate request: %w", err)
		}
		req.Events = []models.CertificateEvent{}
		requests = append(requests, &req)
		requestIDs = append(requestIDs, req.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate certificate requests: %w", err)
	}

	result := &models.PaginatedCertResult{
		Requests: requests,
		Total:    total,
		Limit:    params.Limit,
		Offset:   params.Offset,
		HasMore:  params.Offset+len(requests) < total,
	}

	if len(requests) == 0 {
		return result, nil
	}

	// Fetch all events for the paginated requests
	eventsQuery := `
       SELECT id, certificate_request_id, requester_iss, requester_sub, reviewer_iss, reviewer_sub, new_status, review_notes, created_at
       FROM certificate_events
       WHERE certificate_request_id = ANY($1)
       ORDER BY certificate_request_id, created_at
    `

	eventRows, err := p.pool.Query(ctx, eventsQuery, requestIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for certificate requests: %w", err)
	}
	defer eventRows.Close()

	// CreateUser a map for quick lookup
	requestMap := make(map[int]*models.CertificateRequest)
	for _, req := range requests {
		requestMap[req.ID] = req
	}

	for eventRows.Next() {
		var event models.CertificateEvent
		if err := eventRows.Scan(
			&event.ID,
			&event.CertificateRequestID,
			&event.RequesterIss,
			&event.RequesterSub,
			&event.ReviewerIss,
			&event.ReviewerSub,
			&event.NewStatus,
			&event.ReviewNotes,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if req, ok := requestMap[event.CertificateRequestID]; ok {
			req.Events = append(req.Events, event)
		}
	}

	if err := eventRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate certificate events: %w", err)
	}

	return result, nil
}

func (p *DatabaseProvider) UpdateCertificateRequestStatus(ctx context.Context, requestId int, newStatus models.CertificateRequestStatus, reviewerIss, reviewerSub, notes string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transaction start failed: %w", err)
	}
	defer tx.Rollback(ctx)

	getRequestStatusQuery := `
		SELECT status, owner_iss, owner_sub 
		FROM certificate_requests 
		WHERE id = $1
		`

	var currentStatus models.CertificateRequestStatus
	var requesterIss, requesterSub string
	err = tx.QueryRow(ctx, getRequestStatusQuery, requestId).Scan(&currentStatus, &requesterIss, &requesterSub)
	if err != nil {
		return fmt.Errorf("failed to get current status for certificate request '%d': %w", requestId, err)
	}

	updateStatusQuery := `
		UPDATE certificate_requests
		SET status = $1
		WHERE id = $2
		`

	_, err = tx.Exec(ctx, updateStatusQuery, newStatus, requestId)
	if err != nil {
		return fmt.Errorf("failed to update status for certificate request '%d': %w", requestId, err)
	}

	insertEventQuery := `
		INSERT INTO certificate_events
		(certificate_request_id, requester_iss, requester_sub, reviewer_iss, reviewer_sub, new_status, review_notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

	_, err = tx.Exec(ctx, insertEventQuery, requestId, requesterIss, requesterSub, reviewerIss, reviewerSub, newStatus, notes)
	if err != nil {
		return fmt.Errorf("failed to insert event for certificate request '%d': %w", requestId, err)
	}

	return tx.Commit(ctx)
}

// UpdateCertificateMetadata updates the certificate identifier an metadata
func (p *DatabaseProvider) UpdateCertificateMetadata(ctx context.Context, requestID int, identifier string, metadata map[string]interface{}) error {
	query := `
		UPDATE certificate_requests
		SET certificate_identifier = $1,
			provider_metadata = $2
		WHERE id = $3
	`

	result, err := p.pool.Exec(ctx, query, identifier, metadata, requestID)
	if err != nil {
		return fmt.Errorf("failed to update certificate metadata for request %d: %w", requestID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("certificate request %d not found", requestID)
	}

	return nil
}

// GetApprovedCertificateRequests returns all certificate requests with status = APPROVED
func (p *DatabaseProvider) GetApprovedCertificateRequests(ctx context.Context) ([]*models.CertificateRequest, error) {
	query := `
		SELECT id, owner_iss, owner_sub, message, common_name, dns_names, organizational_units, validity_days, status, requested_at, certificate_identifier, provider_metadata, issued_at, expires_at, serial_number, certificate_pem
		FROM certificate_requests
		WHERE status = $1
		ORDER BY requested_at ASC
	`

	rows, err := p.pool.Query(ctx, query, models.StatusApproved)
	if err != nil {
		return nil, fmt.Errorf("failed to get approved requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.CertificateRequest
	for rows.Next() {
		var req models.CertificateRequest
		if err := rows.Scan(
			&req.ID,
			&req.OwnerIss,
			&req.OwnerSub,
			&req.Message,
			&req.CommonName,
			&req.DNSNames,
			&req.OrganizationalUnits,
			&req.ValidityDays,
			&req.Status,
			&req.RequestedAt,
			&req.CertificateIdentifier,
			&req.ProviderMetadata,
			&req.IssuedAt,
			&req.ExpiresAt,
			&req.SerialNumber,
			&req.CertificatePem,
		); err != nil {
			return nil, fmt.Errorf("failed to scan approved request: %w", err)
		}
		requests = append(requests, &req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate approved requests: %w", err)
	}

	return requests, nil
}

// GetPendingCertificateRequests returns all certificate requests with status = PENDING (awaiting certificate to be ready)
func (p *DatabaseProvider) GetPendingCertificateRequests(ctx context.Context) ([]*models.CertificateRequest, error) {
	query := `
		SELECT id, owner_iss, owner_sub, message, common_name, dns_names, organizational_units, validity_days, status, requested_at, certificate_identifier, provider_metadata, issued_at, expires_at, serial_number, certificate_pem
		FROM certificate_requests
		WHERE status = $1 AND certificate_identifier IS NOT NULL
		ORDER BY requested_at ASC
	`

	rows, err := p.pool.Query(ctx, query, models.StatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.CertificateRequest
	for rows.Next() {
		var req models.CertificateRequest
		if err := rows.Scan(
			&req.ID,
			&req.OwnerIss,
			&req.OwnerSub,
			&req.Message,
			&req.CommonName,
			&req.DNSNames,
			&req.OrganizationalUnits,
			&req.ValidityDays,
			&req.Status,
			&req.RequestedAt,
			&req.CertificateIdentifier,
			&req.ProviderMetadata,
			&req.IssuedAt,
			&req.ExpiresAt,
			&req.SerialNumber,
			&req.CertificatePem,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pending request: %w", err)
		}
		requests = append(requests, &req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pending requests: %w", err)
	}

	return requests, nil
}

// UpdateCertificateRequestIssued updates a certificate request to ISSUED status with the certificate details
func (p *DatabaseProvider) UpdateCertificateRequestIssued(ctx context.Context, requestID int, certPEM, serialNumber string, issuedAt, expiresAt time.Time, systemUserIss, systemUserSub string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transaction start failed: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get request owner for the event
	getRequestQuery := `
		SELECT owner_iss, owner_sub
		FROM certificate_requests
		WHERE id = $1
	`

	var requesterIss, requesterSub string
	err = tx.QueryRow(ctx, getRequestQuery, requestID).Scan(&requesterIss, &requesterSub)
	if err != nil {
		return fmt.Errorf("failed to get request owner: %w", err)
	}

	// Update the request with certificate details
	updateQuery := `
		UPDATE certificate_requests
		SET status = $1,
			certificate_pem = $2,
			serial_number = $3,
			issued_at = $4,
			expires_at = $5
		WHERE id = $6
	`

	result, err := tx.Exec(ctx, updateQuery, models.StatusIssued, certPEM, serialNumber, issuedAt, expiresAt, requestID)
	if err != nil {
		return fmt.Errorf("failed to update certificate to issued: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("certificate request %d not found", requestID)
	}

	// Insert event
	insertEventQuery := `
		INSERT INTO certificate_events
		(certificate_request_id, requester_iss, requester_sub, reviewer_iss, reviewer_sub, new_status, review_notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.Exec(ctx, insertEventQuery, requestID, requesterIss, requesterSub, systemUserIss, systemUserSub, models.StatusIssued, "Certificate issued by cert-manager")
	if err != nil {
		return fmt.Errorf("failed to insert issued event: %w", err)
	}

	return tx.Commit(ctx)
}
