CREATE TABLE users (
    iss          TEXT      NOT NULL,
    sub          TEXT      NOT NULL,
    username     TEXT      NOT NULL,
    display_name TEXT      NOT NULL,
    email        TEXT      NOT NULL,
    is_system    BOOLEAN   NOT NULL DEFAULT FALSE,
    last_logged_in TIMESTAMP,
    created_at   TIMESTAMP,

    PRIMARY KEY (iss, sub)
);

CREATE UNIQUE INDEX idx_users_system ON users(is_system) WHERE is_system = TRUE;

CREATE TABLE user_groups (
    owner_iss  TEXT NOT NULL,
    owner_sub  TEXT NOT NULL,
    group_name TEXT NOT NULL,

    PRIMARY KEY (owner_iss, owner_sub, group_name),
    FOREIGN KEY (owner_iss, owner_sub) REFERENCES users(iss, sub) ON DELETE CASCADE
);

CREATE TABLE service_accounts (
    id          SERIAL PRIMARY KEY,
    iss         TEXT      NOT NULL,
    sub         TEXT      NOT NULL,
    name        TEXT      NOT NULL,

    lookup_id   TEXT      NOT NULL UNIQUE,
    token_hash  TEXT      NOT NULL,
    token_expires_at TIMESTAMP,

    is_disabled BOOLEAN   NOT NULL DEFAULT FALSE,
    deleted_at  TIMESTAMP,

    created_by_iss TEXT   NOT NULL,
    created_by_sub TEXT   NOT NULL,
    created_at  TIMESTAMP,

    CONSTRAINT lookup_id_length CHECK (char_length(lookup_id) >= 16)
);

CREATE UNIQUE INDEX idx_service_accounts_iss_sub   ON service_accounts(iss, sub);
CREATE INDEX idx_service_accounts_token_hash       ON service_accounts(token_hash);
CREATE INDEX idx_service_accounts_owner            ON service_accounts(created_by_iss, created_by_sub);
CREATE INDEX idx_service_accounts_lookup_id        ON service_accounts(lookup_id);
CREATE INDEX idx_service_accounts_deleted_at       ON service_accounts(deleted_at);

CREATE TABLE service_account_scopes (
    owner_iss  TEXT NOT NULL,
    owner_sub  TEXT NOT NULL,
    scope_name TEXT NOT NULL,

    PRIMARY KEY (owner_iss, owner_sub, scope_name),
    FOREIGN KEY (owner_iss, owner_sub) REFERENCES service_accounts(iss, sub) ON DELETE CASCADE
);

CREATE TABLE certificate_requests (
    id          SERIAL PRIMARY KEY NOT NULL,
    owner_iss   TEXT NOT NULL,
    owner_sub   TEXT NOT NULL,
    message     TEXT,

    common_name           TEXT    NOT NULL,
    dns_names             TEXT[]  DEFAULT '{}',
    organizational_units  TEXT[]  DEFAULT '{}',
    validity_days         INTEGER NOT NULL DEFAULT 365,

    status       TEXT      NOT NULL DEFAULT 'pending',
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),

    certificate_identifier TEXT,
    provider_metadata      JSONB,

    issued_at     TIMESTAMP,
    expires_at    TIMESTAMP,
    serial_number TEXT,
    certificate_pem TEXT,

    CONSTRAINT certificate_requests_owner_not_empty CHECK (owner_iss != '' AND owner_sub != '')
);

CREATE INDEX idx_cert_requests_owner      ON certificate_requests(owner_iss, owner_sub);
CREATE INDEX idx_cert_requests_status     ON certificate_requests(status);
CREATE INDEX idx_cert_requests_identifier ON certificate_requests(certificate_identifier);

CREATE TABLE certificate_downloads (
    id                     SERIAL PRIMARY KEY NOT NULL,
    certificate_request_id INTEGER   NOT NULL,
    downloader_sub         TEXT      NOT NULL,
    downloader_iss         TEXT      NOT NULL,

    ip_address   INET NOT NULL,

    user_agent      TEXT,
    browser_name    TEXT,
    browser_version TEXT,
    os_name         TEXT,
    os_version      TEXT,
    device_type     TEXT,

    downloaded_at TIMESTAMP DEFAULT NOW(),

    FOREIGN KEY (certificate_request_id) REFERENCES certificate_requests(id) ON DELETE RESTRICT,
    CONSTRAINT certificate_downloads_downloader_not_empty CHECK (downloader_iss != '' AND downloader_sub != '')
);

CREATE INDEX idx_cert_downloads_owner   ON certificate_downloads(downloader_iss, downloader_sub);
CREATE INDEX idx_cert_downloads_ip      ON certificate_downloads(ip_address);
CREATE INDEX idx_cert_downloads_cert_id ON certificate_downloads(certificate_request_id);

CREATE TABLE certificate_events (
    id                     SERIAL PRIMARY KEY NOT NULL,
    certificate_request_id INTEGER   NOT NULL,
    requester_iss          TEXT      NOT NULL,
    requester_sub          TEXT      NOT NULL,
    reviewer_iss           TEXT      NOT NULL,
    reviewer_sub           TEXT      NOT NULL,

    new_status   TEXT NOT NULL,
    review_notes TEXT,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),

    FOREIGN KEY (certificate_request_id) REFERENCES certificate_requests(id) ON DELETE CASCADE,
    CONSTRAINT certificate_events_requester_not_empty CHECK (requester_iss != '' AND requester_sub != ''),
    CONSTRAINT certificate_events_reviewer_not_empty  CHECK (reviewer_iss  != '' AND reviewer_sub  != '')
);

CREATE INDEX idx_cert_events_request_id ON certificate_events(certificate_request_id);
CREATE INDEX idx_cert_events_requester  ON certificate_events(requester_iss, requester_sub);
CREATE INDEX idx_cert_events_reviewer   ON certificate_events(reviewer_iss, reviewer_sub);

CREATE TABLE issued_certificates (
    identifier TEXT PRIMARY KEY,
    cert_pem   TEXT      NOT NULL,
    key_pem    TEXT      NOT NULL,
    ca_pem     TEXT      NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE certificate_download_tokens (
    token_hash             TEXT PRIMARY KEY,
    certificate_request_id INTEGER   NOT NULL,
    principal_iss          TEXT      NOT NULL,
    principal_sub          TEXT      NOT NULL,
    passphrase             TEXT      NOT NULL,
    created_at             TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at             TIMESTAMP NOT NULL,
    used_at                TIMESTAMP,

    FOREIGN KEY (certificate_request_id) REFERENCES certificate_requests(id) ON DELETE CASCADE
);

CREATE INDEX idx_download_tokens_expires ON certificate_download_tokens(expires_at);
