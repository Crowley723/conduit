package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
)

// AddIPToWhitelist adds a firewall ip whitelist entry.
// This function uses a transaction to atomically check limits and insert the entry,
// preventing race conditions where multiple concurrent requests could exceed limits.
func (p *DatabaseProvider) AddIPToWhitelist(ctx context.Context, ownerIss, ownerSub, aliasName, aliasUUID, ipAddress, description string, expiresAt *time.Time, clientIP, userAgent *string) (*models.FirewallIPWhitelistEntry, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check user limit within transaction (prevents race condition)
	userCountQuery := `
		SELECT COUNT(*)
		FROM firewall_ip_whitelist_entries
		WHERE owner_iss = $1 AND owner_sub = $2 AND alias_uuid = $3
		  AND status IN ('requested', 'added')
	`
	var userCount int
	err = tx.QueryRow(ctx, userCountQuery, ownerIss, ownerSub, aliasUUID).Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count user active IPs: %w", err)
	}

	totalCountQuery := `
		SELECT COUNT(*)
		FROM firewall_ip_whitelist_entries
		WHERE alias_uuid = $1
		  AND status IN ('requested', 'added')
	`
	var totalCount int
	err = tx.QueryRow(ctx, totalCountQuery, aliasUUID).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count total active IPs: %w", err)
	}

	duplicateCheckQuery := `
		SELECT id
		FROM firewall_ip_whitelist_entries
		WHERE owner_iss = $1 AND owner_sub = $2 AND alias_uuid = $3 AND ip_address = $4
		  AND status IN ('requested', 'added')
		LIMIT 1
	`
	var existingID int
	err = tx.QueryRow(ctx, duplicateCheckQuery, ownerIss, ownerSub, aliasUUID, ipAddress).Scan(&existingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check for duplicate IP: %w", err)
	}
	if err == nil {
		// Duplicate found
		return nil, fmt.Errorf("you already have this IP address whitelisted for this alias")
	}

	insertQuery := `
		INSERT INTO firewall_ip_whitelist_entries (owner_iss, owner_sub, alias_name, alias_uuid, ip_address, description, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var recordId int
	err = tx.QueryRow(ctx, insertQuery,
		ownerIss, ownerSub, aliasName, aliasUUID, ipAddress, description, expiresAt).Scan(&recordId)
	if err != nil {
		return nil, fmt.Errorf("failed to add IP to whitelist: %w", err)
	}

	eventQuery := `
		INSERT INTO firewall_whitelist_events (whitelist_id, actor_iss, actor_sub, event_type, notes, client_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.Exec(ctx, eventQuery, recordId, ownerIss, ownerSub, "requested", nil, clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to create requested event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return p.GetWhitelistEntryByID(ctx, recordId)
}

// GetWhitelistEntryByID returns a firewall whitelist entry, including events for a specific id.
func (p *DatabaseProvider) GetWhitelistEntryByID(ctx context.Context, id int) (*models.FirewallIPWhitelistEntry, error) {
	query := `
        SELECT id, owner_iss, owner_sub, alias_name, alias_uuid, ip_address::text, ip_version, description, status, 
               requested_at, added_at, removed_at, expires_at, removed_by_iss, removed_by_sub, removal_reason
        FROM firewall_ip_whitelist_entries
        WHERE id = $1
    `

	var whitelistEntry models.FirewallIPWhitelistEntry
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&whitelistEntry.ID,
		&whitelistEntry.OwnerIss,
		&whitelistEntry.OwnerSub,
		&whitelistEntry.AliasName,
		&whitelistEntry.AliasUUID,
		&whitelistEntry.IPAddress,
		&whitelistEntry.IPVersion,
		&whitelistEntry.Description,
		&whitelistEntry.Status,
		&whitelistEntry.RequestedAt,
		&whitelistEntry.AddedAt,
		&whitelistEntry.RemovedAt,
		&whitelistEntry.ExpiresAt,
		&whitelistEntry.RemovedByIss,
		&whitelistEntry.RemovedBySub,
		&whitelistEntry.RemovalReason,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("whitelist entry not found")
		}
		return nil, fmt.Errorf("failed to get whitelist entry with id '%d': %w", id, err)
	}

	events, err := p.GetWhitelistEventsByEntry(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for whitelist entry '%d': %w", id, err)
	}

	whitelistEntry.Events = make([]models.FirewallIPWhitelistEvent, len(events))
	for i, event := range events {
		whitelistEntry.Events[i] = *event
	}

	return &whitelistEntry, nil
}

func (p *DatabaseProvider) GetUserWhitelistEntries(ctx context.Context, ownerIss, ownerSub string) ([]*models.FirewallIPWhitelistEntry, error) {
	// Single query with JOIN to get all entries and events in one go (FIXES N+1)
	query := `
		SELECT
			fiwe.id, fiwe.owner_iss, fiwe.owner_sub, fiwe.alias_name, fiwe.alias_uuid,
			fiwe.ip_address::text, fiwe.ip_version, fiwe.description, fiwe.status,
			fiwe.requested_at, fiwe.added_at, fiwe.removed_at, fiwe.expires_at,
			fiwe.removed_by_iss, fiwe.removed_by_sub, fiwe.removal_reason,
			owner.username as owner_username,
			owner.display_name as owner_display_name,
			fwe.id as event_id,
			fwe.whitelist_id as event_whitelist_id,
			fwe.actor_iss as event_actor_iss,
			fwe.actor_sub as event_actor_sub,
			fwe.event_type,
			fwe.notes as event_notes,
			fwe.client_ip::text as event_client_ip,
			fwe.user_agent as event_user_agent,
			fwe.created_at as event_created_at,
			COALESCE(event_actor.username, sa_creator.username) as event_actor_username,
			COALESCE(event_actor.display_name, sa_creator.display_name) as event_actor_display_name
		FROM firewall_ip_whitelist_entries fiwe
		JOIN users owner ON fiwe.owner_iss = owner.iss AND fiwe.owner_sub = owner.sub
		LEFT JOIN firewall_whitelist_events fwe ON fiwe.id = fwe.whitelist_id
		LEFT JOIN users event_actor ON fwe.actor_iss = event_actor.iss AND fwe.actor_sub = event_actor.sub
		LEFT JOIN service_accounts sa ON fwe.actor_iss = sa.iss AND fwe.actor_sub = sa.sub
		LEFT JOIN users sa_creator ON sa.created_by_iss = sa_creator.iss AND sa.created_by_sub = sa_creator.sub
		WHERE fiwe.owner_iss = $1 AND fiwe.owner_sub = $2
		ORDER BY fiwe.requested_at DESC, fwe.created_at DESC
	`

	rows, err := p.pool.Query(ctx, query, ownerIss, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("failed to get user whitelist entries: %w", err)
	}
	defer rows.Close()

	entriesMap := make(map[int]*models.FirewallIPWhitelistEntry)
	var entryOrder []int

	for rows.Next() {
		var (
			entryID, ipVersion                                  int
			ownerIss, ownerSub, aliasName, aliasUUID, ipAddress string
			description, status                                 string
			ownerUsername, ownerDisplayName                     string
			requestedAt                                         time.Time
			addedAt, removedAt, expiresAt                       *time.Time
			removedByIss, removedBySub, removalReason           *string
			eventID, eventWhitelistID                           *int
			eventActorIss, eventActorSub, eventType, eventNotes *string
			eventClientIP, eventUserAgent                       *string
			eventCreatedAt                                      *time.Time
			eventActorUsername, eventActorDisplay               *string
		)

		err := rows.Scan(
			&entryID, &ownerIss, &ownerSub, &aliasName, &aliasUUID,
			&ipAddress, &ipVersion, &description, &status,
			&requestedAt, &addedAt, &removedAt, &expiresAt,
			&removedByIss, &removedBySub, &removalReason,
			&ownerUsername, &ownerDisplayName,
			&eventID, &eventWhitelistID,
			&eventActorIss, &eventActorSub, &eventType, &eventNotes,
			&eventClientIP, &eventUserAgent, &eventCreatedAt,
			&eventActorUsername, &eventActorDisplay,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan whitelist entry: %w", err)
		}

		entry, exists := entriesMap[entryID]
		if !exists {
			entry = &models.FirewallIPWhitelistEntry{
				ID:               entryID,
				OwnerIss:         ownerIss,
				OwnerSub:         ownerSub,
				OwnerUsername:    ownerUsername,
				OwnerDisplayName: ownerDisplayName,
				AliasName:        aliasName,
				AliasUUID:        aliasUUID,
				IPAddress:        ipAddress,
				IPVersion:        ipVersion,
				Description:      description,
				Status:           models.FirewallIPWhitelistStatus(status),
				RequestedAt:      requestedAt,
				AddedAt:          addedAt,
				RemovedAt:        removedAt,
				ExpiresAt:        expiresAt,
				RemovedByIss:     removedByIss,
				RemovedBySub:     removedBySub,
				RemovalReason:    removalReason,
				Events:           []models.FirewallIPWhitelistEvent{},
			}
			entriesMap[entryID] = entry
			entryOrder = append(entryOrder, entryID)
		}

		if eventID != nil {
			event := models.FirewallIPWhitelistEvent{
				ID:               *eventID,
				WhitelistID:      *eventWhitelistID,
				ActorISS:         *eventActorIss,
				ActorSub:         *eventActorSub,
				EventType:        *eventType,
				Notes:            eventNotes,
				ClientIP:         eventClientIP,
				UserAgent:        eventUserAgent,
				CreatedAt:        *eventCreatedAt,
				ActorUsername:    *eventActorUsername,
				ActorDisplayName: *eventActorDisplay,
			}
			entry.Events = append(entry.Events, event)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate whitelist entries: %w", err)
	}

	var entries []*models.FirewallIPWhitelistEntry
	for _, id := range entryOrder {
		entries = append(entries, entriesMap[id])
	}

	return entries, nil
}

func (p *DatabaseProvider) GetAllWhitelistEntries(ctx context.Context) ([]*models.FirewallIPWhitelistEntry, error) {
	// CRITICAL FIX: Previous version was missing WHERE clause on events query,
	// which caused ALL events from database to be returned for every entry
	// Single query with JOIN to get all entries and events in one go (FIXES N+1)
	query := `
		SELECT
			fiwe.id, fiwe.owner_iss, fiwe.owner_sub, fiwe.alias_name, fiwe.alias_uuid,
			fiwe.ip_address::text, fiwe.ip_version, fiwe.description, fiwe.status,
			fiwe.requested_at, fiwe.added_at, fiwe.removed_at, fiwe.expires_at,
			fiwe.removed_by_iss, fiwe.removed_by_sub, fiwe.removal_reason,
			owner.username as owner_username,
			owner.display_name as owner_display_name,
			fwe.id as event_id,
			fwe.whitelist_id as event_whitelist_id,
			fwe.actor_iss as event_actor_iss,
			fwe.actor_sub as event_actor_sub,
			fwe.event_type,
			fwe.notes as event_notes,
			fwe.client_ip::text as event_client_ip,
			fwe.user_agent as event_user_agent,
			fwe.created_at as event_created_at,
			COALESCE(event_actor.username, sa_creator.username) as event_actor_username,
			COALESCE(event_actor.display_name, sa_creator.display_name) as event_actor_display_name
		FROM firewall_ip_whitelist_entries fiwe
		JOIN users owner ON fiwe.owner_iss = owner.iss AND fiwe.owner_sub = owner.sub
		LEFT JOIN firewall_whitelist_events fwe ON fiwe.id = fwe.whitelist_id
		LEFT JOIN users event_actor ON fwe.actor_iss = event_actor.iss AND fwe.actor_sub = event_actor.sub
		LEFT JOIN service_accounts sa ON fwe.actor_iss = sa.iss AND fwe.actor_sub = sa.sub
		LEFT JOIN users sa_creator ON sa.created_by_iss = sa_creator.iss AND sa.created_by_sub = sa_creator.sub
		ORDER BY fiwe.requested_at DESC, fwe.created_at DESC
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all whitelist entries: %w", err)
	}
	defer rows.Close()

	entriesMap := make(map[int]*models.FirewallIPWhitelistEntry)
	var entryOrder []int

	for rows.Next() {
		var (
			entryID, ipVersion                                  int
			ownerIss, ownerSub, aliasName, aliasUUID, ipAddress string
			description, status                                 string
			ownerUsername, ownerDisplayName                     string
			requestedAt                                         time.Time
			addedAt, removedAt, expiresAt                       *time.Time
			removedByIss, removedBySub, removalReason           *string
			eventID, eventWhitelistID                           *int
			eventActorIss, eventActorSub, eventType, eventNotes *string
			eventClientIP, eventUserAgent                       *string
			eventCreatedAt                                      *time.Time
			eventActorUsername, eventActorDisplay               *string
		)

		err := rows.Scan(
			&entryID, &ownerIss, &ownerSub, &aliasName, &aliasUUID,
			&ipAddress, &ipVersion, &description, &status,
			&requestedAt, &addedAt, &removedAt, &expiresAt,
			&removedByIss, &removedBySub, &removalReason,
			&ownerUsername, &ownerDisplayName,
			&eventID, &eventWhitelistID,
			&eventActorIss, &eventActorSub, &eventType, &eventNotes,
			&eventClientIP, &eventUserAgent, &eventCreatedAt,
			&eventActorUsername, &eventActorDisplay,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan whitelist entry: %w", err)
		}

		entry, exists := entriesMap[entryID]
		if !exists {
			entry = &models.FirewallIPWhitelistEntry{
				ID:               entryID,
				OwnerIss:         ownerIss,
				OwnerSub:         ownerSub,
				OwnerUsername:    ownerUsername,
				OwnerDisplayName: ownerDisplayName,
				AliasName:        aliasName,
				AliasUUID:        aliasUUID,
				IPAddress:        ipAddress,
				IPVersion:        ipVersion,
				Description:      description,
				Status:           models.FirewallIPWhitelistStatus(status),
				RequestedAt:      requestedAt,
				AddedAt:          addedAt,
				RemovedAt:        removedAt,
				ExpiresAt:        expiresAt,
				RemovedByIss:     removedByIss,
				RemovedBySub:     removedBySub,
				RemovalReason:    removalReason,
				Events:           []models.FirewallIPWhitelistEvent{},
			}
			entriesMap[entryID] = entry
			entryOrder = append(entryOrder, entryID)
		}

		if eventID != nil {
			event := models.FirewallIPWhitelistEvent{
				ID:               *eventID,
				WhitelistID:      *eventWhitelistID,
				ActorISS:         *eventActorIss,
				ActorSub:         *eventActorSub,
				EventType:        *eventType,
				Notes:            eventNotes,
				ClientIP:         eventClientIP,
				UserAgent:        eventUserAgent,
				CreatedAt:        *eventCreatedAt,
				ActorUsername:    *eventActorUsername,
				ActorDisplayName: *eventActorDisplay,
			}
			entry.Events = append(entry.Events, event)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate whitelist entries: %w", err)
	}

	var entries []*models.FirewallIPWhitelistEntry
	for _, id := range entryOrder {
		entries = append(entries, entriesMap[id])
	}

	return entries, nil
}

// RemoveIPFromWhitelist removes an IP from the whitelist
func (p *DatabaseProvider) RemoveIPFromWhitelist(ctx context.Context, id int, ownerIss, ownerSub string, clientIP, userAgent *string) error {
	query := `
        UPDATE firewall_ip_whitelist_entries
        SET status = 'removed', removed_at = NOW()
        WHERE id = $1
    `

	result, err := p.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to remove IP from whitelist: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("whitelist entry not found")
	}

	err = p.CreateWhitelistEvent(ctx, id, ownerIss, ownerSub, "removed", "", clientIP, userAgent)
	if err != nil {
		return fmt.Errorf("failed to create removal event: %w", err)
	}

	return nil
}

// BlacklistIP blacklists an IP address (prevents re-adding)
func (p *DatabaseProvider) BlacklistIP(ctx context.Context, id int, adminIss, adminSub, reason string) error {
	query := `
        UPDATE firewall_ip_whitelist_entries
        SET status = 'blacklisted_by_admin',
            removed_at = NOW(),
            removed_by_iss = $2,
            removed_by_sub = $3,
            removal_reason = $4
        WHERE id = $1
    `

	result, err := p.pool.Exec(ctx, query, id, adminIss, adminSub, reason)
	if err != nil {
		return fmt.Errorf("failed to blacklist IP: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("whitelist entry not found")
	}

	err = p.CreateWhitelistEvent(ctx, id, adminIss, adminSub, "blacklisted_by_admin", reason, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create blacklist event: %w", err)
	}

	return nil
}

// BlacklistIPAddress blacklists ALL entries with a specific IP address for an alias.
// Returns the count of entries affected.
func (p *DatabaseProvider) BlacklistIPAddress(ctx context.Context, aliasUUID, ipAddress, adminIss, adminSub, reason string) (int, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get all entries with this IP for this alias
	selectQuery := `
		SELECT id
		FROM firewall_ip_whitelist_entries
		WHERE alias_uuid = $1
		  AND ip_address = $2
		  AND status != 'blacklisted_by_admin'
	`

	rows, err := tx.Query(ctx, selectQuery, aliasUUID, ipAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to get entries to blacklist: %w", err)
	}
	defer rows.Close()

	var entryIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("failed to scan entry ID: %w", err)
		}
		entryIDs = append(entryIDs, id)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to iterate entries: %w", err)
	}

	if len(entryIDs) == 0 {
		return 0, fmt.Errorf("no entries found to blacklist")
	}

	// Update all entries to blacklisted status
	updateQuery := `
		UPDATE firewall_ip_whitelist_entries
		SET status = 'blacklisted_by_admin',
		    removed_at = NOW(),
		    removed_by_iss = $2,
		    removed_by_sub = $3,
		    removal_reason = $4
		WHERE id = ANY($1)
	`

	_, err = tx.Exec(ctx, updateQuery, entryIDs, adminIss, adminSub, reason)
	if err != nil {
		return 0, fmt.Errorf("failed to update entries to blacklisted: %w", err)
	}

	// Create audit events for each blacklisted entry
	for _, id := range entryIDs {
		eventQuery := `
			INSERT INTO firewall_whitelist_events (whitelist_id, actor_iss, actor_sub, event_type, notes, client_ip, user_agent)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		_, err = tx.Exec(ctx, eventQuery, id, adminIss, adminSub, "blacklisted_by_admin", reason, nil, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create blacklist event for entry %d: %w", id, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return len(entryIDs), nil
}

// IsIPBlacklisted checks if an IP is blacklisted for an alias
func (p *DatabaseProvider) IsIPBlacklisted(ctx context.Context, aliasUUID, ipAddress string) (bool, error) {
	query := `
        SELECT EXISTS(
            SELECT 1 FROM firewall_ip_whitelist_entries
            WHERE alias_uuid = $1 
              AND ip_address = $2 
              AND status = 'blacklisted_by_admin'
        )
    `

	var exists bool
	err := p.pool.QueryRow(ctx, query, aliasUUID, ipAddress).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if IP is blacklisted: %w", err)
	}

	return exists, nil
}

// GetPendingIPs gets all IPs that need to be added to the firewall for a specific alias
func (p *DatabaseProvider) GetPendingIPs(ctx context.Context, aliasUUID string) ([]*models.FirewallIPWhitelistEntry, error) {
	query := `
        SELECT id, owner_iss, owner_sub, alias_name, alias_uuid, ip_address::text, ip_version, description, status, 
               requested_at, added_at, removed_at, expires_at, removed_by_iss, removed_by_sub, removal_reason
        FROM firewall_ip_whitelist_entries
        WHERE alias_uuid = $1 
          AND status = 'requested'
          AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY requested_at ASC
    `

	rows, err := p.pool.Query(ctx, query, aliasUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending IPs: %w", err)
	}
	defer rows.Close()

	var entries []*models.FirewallIPWhitelistEntry
	for rows.Next() {
		var entry models.FirewallIPWhitelistEntry
		err := rows.Scan(
			&entry.ID,
			&entry.OwnerIss,
			&entry.OwnerSub,
			&entry.AliasName,
			&entry.AliasUUID,
			&entry.IPAddress,
			&entry.IPVersion,
			&entry.Description,
			&entry.Status,
			&entry.RequestedAt,
			&entry.AddedAt,
			&entry.RemovedAt,
			&entry.ExpiresAt,
			&entry.RemovedByIss,
			&entry.RemovedBySub,
			&entry.RemovalReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending IP: %w", err)
		}
		entry.Events = []models.FirewallIPWhitelistEvent{}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pending IPs: %w", err)
	}

	return entries, nil
}

// MarkIPsAsAdded marks IPs as successfully added to the firewall
func (p *DatabaseProvider) MarkIPsAsAdded(ctx context.Context, ids []int, systemUserIss, systemUserSub string) error {
	query := `
        UPDATE firewall_ip_whitelist_entries
        SET status = 'added', added_at = NOW()
        WHERE id = ANY($1) AND status = 'requested'
    `

	_, err := p.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("failed to mark IPs as added: %w", err)
	}

	for _, id := range ids {
		err = p.CreateWhitelistEvent(ctx, id, systemUserIss, systemUserSub, "added", "", nil, nil)
		if err != nil {
			return fmt.Errorf("failed to create added event for IP %d: %w", id, err)
		}
	}

	return nil
}

// ExpireOldIPs marks expired IPs as removed and returns the count of expired IPs
func (p *DatabaseProvider) ExpireOldIPs(ctx context.Context, systemUserIss, systemUserSub string) (int, error) {
	selectQuery := `
        SELECT id
        FROM firewall_ip_whitelist_entries
        WHERE status IN ('requested', 'added')
          AND expires_at IS NOT NULL
          AND expires_at <= NOW()
    `

	rows, err := p.pool.Query(ctx, selectQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to get expired IPs: %w", err)
	}
	defer rows.Close()

	var expiredIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("failed to scan expired IP ID: %w", err)
		}
		expiredIDs = append(expiredIDs, id)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to iterate expired IPs: %w", err)
	}

	if len(expiredIDs) == 0 {
		return 0, nil
	}

	updateQuery := `
        UPDATE firewall_ip_whitelist_entries
        SET status = 'removed', removed_at = NOW()
        WHERE id = ANY($1)
    `

	_, err = p.pool.Exec(ctx, updateQuery, expiredIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to mark IPs as expired: %w", err)
	}

	for _, id := range expiredIDs {
		err = p.CreateWhitelistEvent(ctx, id, systemUserIss, systemUserSub, "expired", "", nil, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create expired event for IP %d: %w", id, err)
		}
	}

	return len(expiredIDs), nil
}

// CountUserActiveIPs counts how many active IPs a user has for a specific alias
func (p *DatabaseProvider) CountUserActiveIPs(ctx context.Context, ownerIss, ownerSub, aliasUUID string) (int, error) {
	query := `
        SELECT COUNT(*) 
        FROM firewall_ip_whitelist_entries
        WHERE owner_iss = $1 AND owner_sub = $2 AND alias_uuid = $3 
          AND status IN ('requested', 'added')
    `

	var count int
	err := p.pool.QueryRow(ctx, query, ownerIss, ownerSub, aliasUUID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user active IPs: %w", err)
	}

	return count, nil
}

// CountTotalActiveIPs counts total active IPs for a specific alias
func (p *DatabaseProvider) CountTotalActiveIPs(ctx context.Context, aliasUUID string) (int, error) {
	query := `
        SELECT COUNT(*) 
        FROM firewall_ip_whitelist_entries
        WHERE alias_uuid = $1 
          AND status IN ('requested', 'added')
    `

	var count int
	err := p.pool.QueryRow(ctx, query, aliasUUID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count total active IPs: %w", err)
	}

	return count, nil
}
