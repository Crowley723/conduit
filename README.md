# Conduit

A web-based mTLS certificate management dashboard. Conduit provides a self-hosted interface for requesting, issuing, and downloading client certificates backed by Kubernetes cert-manager, with OIDC authentication and full audit logging.

## Purpose

Conduit is designed for teams and individuals who need a simple, secure way to manage mTLS client certificates without direct cluster access:

- **Certificate Requests**: Users request certificates through a web UI; issued certificates are downloadable as bundles
- **Kubernetes-backed Issuance**: Integrates with cert-manager for certificate lifecycle management
- **Service Accounts**: Non-human clients can authenticate and request certificates via service accounts
- **Audit Logging**: All certificate operations are recorded for compliance and visibility

## Architecture

**Backend**: Go HTTP server with OIDC authentication
- Certificate request and download handlers
- Kubernetes cert-manager integration for issuance
- PostgreSQL for certificate records, users, service accounts, and audit log
- Redis-backed session management
- Background jobs for certificate creation and status polling
- Leader election for distributed deployments

**Frontend**: React application with TypeScript
- TanStack Router for navigation
- React Query for data management
- Tailwind CSS / shadcn-ui for styling

## Key Features

- OIDC authentication
- Certificate request and download via web UI
- Service account support for programmatic access
- cert-manager integration (Kubernetes)
- Audit log for all certificate operations
- Docker and Helm deployment ready

## Development

```bash
# Install dependencies
make install

# Run full stack (backend + frontend)
make dev

# Run tests
make test

# Docker Compose
make dev-docker
```

## Deployment

The application is containerized and includes Helm charts for Kubernetes deployment.

## TODO

- [ ] **CLI tool**
  - [ ] Cleanup expired download tokens
  - [ ] Cleanup expired certificates
  - [ ] Cleanup old certificate download logs
