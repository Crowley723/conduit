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

CREATE TABLE issued_certificates (
    identifier TEXT PRIMARY KEY,
    cert_pem TEXT NOT NULL,
    key_pem TEXT NOT NULL,
    ca_pem TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);