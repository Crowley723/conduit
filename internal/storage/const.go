package storage

import (
	"errors"
)

const (
	SystemUsername    = "system"
	SystemDisplayName = "System"
	SystemEmail       = "noreply@system"
	SystemSub         = "system"
)

var (
	CertificateRequestNotFoundError  = errors.New("certificate request not found")
	DownloadTokenNotFoundOrExpired   = errors.New("download token not found or expired")
)
