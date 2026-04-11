DROP INDEX IF EXISTS idx_cert_requests_k8s_name;

ALTER TABLE certificate_requests
    ADD COLUMN certificate_identifier TEXT,
    ADD COLUMN provider_metadata JSONB;
UPDATE certificate_requests
SET certificate_identifier = k8s_certificate_name,
    provider_metadata = jsonb_build_object(
        'namespace', k8s_namespace,
        'secret_name', k8s_secret_name
    )
WHERE k8s_certificate_name IS NOT NULL;

ALTER TABLE certificate_requests
    DROP COLUMN k8s_certificate_name,
    DROP COLUMN k8s_namespace,
    DROP COLUMN k8s_secret_name;

CREATE INDEX idx_cert_requests_identifier ON certificate_requests(certificate_identifier);

CREATE TABLE certificate_authority (
    id SERIAL PRIMARY KEY,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    cert_pem BYTEA NOT NULL,
    key_pem BYTEA NOT NULL,
    ca_pem BYTEA NOT NULL,

    common_name TEXT NOT NULL,
    organization TEXT,
    country TEXT,
    locality TEXT,
    province TEXT,

    serial_number TEXT NOT NULL,
    key_algorithm TEXT NOT NULL,

    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE (is_active)
);

CREATE TABLE issued_certificates (
    identifier TEXT PRIMARY KEY,
    cert_pem BYTEA NOT NULL,
    key_pem BYTEA,
    ca_pem BYTEA NOT NULL,

    common_name TEXT NOT NULL,
    organization TEXT,
    country TEXT,
    locality TEXT,
    province TEXT,

    dns_names TEXT[] DEFAULT '{}',
    organizational_units TEXT[] DEFAULT '{}',
    serial_number TEXT NOT NULL,
    key_algorithm TEXT NOT NULL,

    certificate_request_id INTEGER NOT NULL,

    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    FOREIGN KEY (certificate_request_id) REFERENCES certificate_requests(id) ON DELETE RESTRICT
);

CREATE INDEX idx_issued_certs_request_id ON issued_certificates(certificate_request_id);
CREATE INDEX idx_issued_certs_expires_at ON issued_certificates(expires_at);

CREATE TABLE encryption (
    id SERIAL PRIMARY KEY,
    validation_data BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

