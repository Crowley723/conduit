package models

import (
	"net/netip"
	"time"
)

type CertificateDownload struct {
	ID                   int
	CertificateRequestID int
	Sub                  string
	Iss                  string
	IPAddress            netip.Addr
	UserAgent            string
	DownloadedAt         time.Time
}
