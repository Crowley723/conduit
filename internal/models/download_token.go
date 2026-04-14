package models

import "time"

type DownloadToken struct {
	TokenHash     string
	CertificateID int
	PrincipalIss  string
	PrincipalSub  string
	Passphrase    string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	UsedAt        *time.Time
}
