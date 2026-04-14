package storage

import (
	"context"
	"log/slog"
	"time"

	"github.com/Crowley723/conduit/internal/models"

	"github.com/avct/uasurfer"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -source=storage.go -destination=../mocks/storage.go -package=mocks -mock_names=Provider=MockStorageProvider

// noinspection GoNameStartsWithPackageName
type Provider interface {
	GetPool() *pgxpool.Pool
	Close()
	Ping(ctx context.Context) error
	RunMigrations(ctx context.Context) error

	EnsureSystemUser(ctx context.Context, logger *slog.Logger) error
	GetSystemUser(ctx context.Context) (iss, sub string, err error)

	CreateUser(ctx context.Context, sub, iss, username, displayName, email string) (*models.User, error)
	UpsertUser(ctx context.Context, sub, iss, username, displayName, email string, groups []string) (*models.User, error)
	GetUserByID(ctx context.Context, iss, sub string) (*models.User, error)

	/* Certificate Request Queries */

	CreateCertificateRequest(ctx context.Context, sub string, iss string, commonName string, status string, message string, dnsNames []string, organizationalUnits []string, validityDays int) (*models.CertificateRequest, error)
	GetCertificateRequestByID(ctx context.Context, id int) (*models.CertificateRequest, error)
	GetCertificateRequests(ctx context.Context) ([]*models.CertificateRequest, error)
	GetCertificateRequestsByUser(ctx context.Context, sub string, iss string) ([]*models.CertificateRequest, error)
	GetCertificateRequestsPaginated(ctx context.Context, params models.PaginationParams) (*models.PaginatedCertResult, error)
	UpdateCertificateRequestStatus(ctx context.Context, requestId int, newStatus models.CertificateRequestStatus, reviewerIss string, reviewerSub string, notes string) error
	UpdateCertificateMetadata(ctx context.Context, requestID int, identifier string, metadata map[string]interface{}) error
	GetApprovedCertificateRequests(ctx context.Context) ([]*models.CertificateRequest, error)
	GetPendingCertificateRequests(ctx context.Context) ([]*models.CertificateRequest, error)
	UpdateCertificateRequestIssued(ctx context.Context, requestID int, certPEM string, serialNumber string, issuedAt time.Time, expiresAt time.Time, systemUserIss string, systemUserSub string) error

	/* Service Account Queries */

	CreateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error)
	GetServiceAccountByID(ctx context.Context, iss string, sub string) (*models.ServiceAccount, error)
	GetServiceAccountByLookupId(ctx context.Context, tokenHash string) (*models.ServiceAccount, error)
	GetServiceAccountsByCreator(ctx context.Context, iss string, sub string) ([]*models.ServiceAccount, error)
	PauseServiceAccount(ctx context.Context, iss string, sub string) error
	UnpauseServiceAccount(ctx context.Context, iss, sub string) error
	DeleteServiceAccount(ctx context.Context, iss string, sub string) error
	DisableServiceAccount(ctx context.Context, iss string, sub string) error
	EnableServiceAccount(ctx context.Context, iss, sub string) error

	/* Audit Log Queries */

	InsertAuditLogCertificateDownload(ctx context.Context, certId int, sub, iss, ipAddress, rawUserAgent string, userAgent uasurfer.UserAgent) (*models.CertificateDownload, error)
	GetCertificateDownloadAuditLogByID(ctx context.Context, id int) (*models.CertificateDownload, error)
	GetRecentCertificateDownloadLogs(ctx context.Context, limit int) ([]models.CertificateDownload, error)
}
