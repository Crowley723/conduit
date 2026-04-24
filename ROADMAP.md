# Conduit Roadmap

Conduit started as a personal homelab tool and is being rebuilt as a focused, self-hostable mTLS certificate management solution. The goal is a stable v1.0 that anyone can deploy as a binary or container, no cluster access required.

---

## Short Term: Pre-Release

**UI**
- [ ] Streamline the frontend around certificate management as the core workflow

**Authentication & Authorization**
- [ ] Internal users: username/password auth for environments without an OIDC provider
- [ ] Multiple OIDC providers: data model already supports `iss/sub` binding, main work is the login/routing flow

**CLI**
- [ ] `conduit login`: browser-based auth that mints a service account token for CLI use
- [ ] Certificate actions: request, view, approve, download, delete
- [ ] Service account management: create, list, revoke
- [ ] `conduit status`: verify server connectivity and auth

**Deployment**
- [ ] Clean binary builds with embedded frontend assets
- [ ] Container images with proper tagging (GHCR)
- [ ] Example configs for common deployment scenarios
- [ ] Installation, configuration, and deployment documentation

---

## Medium Term: Operational Maturity

**Administrative Tools**
- [ ] Database pruning via CLI: expired download tokens, expired certificates, old download logs, configurable retention

**Testing**
- [ ] Integration test suite
- [ ] Separate suite for cert-manager backend
- [ ] End-to-end certificate request and issuance flows

**Certificate Backends**
- [ ] Additional issuers: HashiCorp Vault, Step-CA
- [ ] Multiple CAs simultaneously, with per-request CA selection and CA-specific TTL policies

**Observability & Security**
- [ ] Rate limiting on API endpoints
- [ ] Structured logging improvements

---

## Long Term: Forward Auth Service

The forward auth service would let Conduit act as an authorization layer for reverse proxies, parsing mTLS certificate identity from request headers and making ACL-based allow/deny decisions.

**Phase 1: Integrated**
- [ ] Forward auth endpoint
- [ ] Parse client cert details from proxy headers
- [ ] Identity verification against certificate database
- [ ] ACL-based authz with audit logging

**Phase 2: Standalone Service**
- [ ] Separate deployable service with shared database access
- [ ] Redis/in-memory caching to minimize DB round trips
- [ ] Horizontal scaling support

---

## v1.0 Release Criteria

- [ ] Binary and container distributions
- [ ] Installation and configuration documentation
- [ ] Internal user authentication
- [ ] Multiple OIDC provider support
- [ ] CLI with certificate actions and service account management
- [ ] Integration test suite passing
- [ ] Authentication flow security review
- [ ] Helm charts and example Kubernetes manifests
- [ ] Migration guide from pre-1.0