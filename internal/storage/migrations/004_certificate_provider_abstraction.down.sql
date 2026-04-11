DROP TABLE IF EXISTS encryption;

DROP INDEX IF EXISTS idx_issued_certs_expires_at;
DROP INDEX IF EXISTS idx_issued_certs_request_id;

DROP TABLE IF EXISTS issued_certificates;
DROP TABLE IF EXISTS certificate_authority;

DROP INDEX IF EXISTS idx_cert_requests_identifier;

ALTER TABLE certificate_requests
    ADD COLUMN k8s_certificate_name TEXT,
    ADD COLUMN k8s_namespace TEXT,
    ADD COLUMN k8s_secret_name TEXT;

UPDATE certificate_requests
SET k8s_certificate_name = certificate_identifier,
    k8s_namespace = provider_metadata->>'namespace',
    k8s_secret_name = provider_metadata->>'secret_name'
WHERE certificate_identifier IS NOT NULL;

ALTER TABLE certificate_requests
    DROP COLUMN certificate_identifier,
    DROP COLUMN provider_metadata;

CREATE INDEX idx_cert_requests_k8s_name ON certificate_requests(k8s_certificate_name, k8s_namespace);
