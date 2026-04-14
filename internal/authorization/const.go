package authorization

const (
	ScopeMTLSRequestCert  = "mtls:request"
	ScopeMTLSReadCert     = "mtls:read"
	ScopeMTLSApproveCert  = "mtls:approve"
	ScopeMTLSRenewCert    = "mtls:renew"
	ScopeMTLSRevokeCert   = "mtls:revoke"
	ScopeMTLSDownloadCert = "mtls:download"
)

const (
	ScopeMTLSDownloadAllCerts = "mtls:download_all"
	ScopeMTLSReadAllCerts     = "mtls:read_all"
	ScopeMTLSAutoApproveCert  = "mtls:auto_approve"
	ScopeMTLSSelfApproveCerts = "mtls:self_approve_certs"
)

// GetAllValidScopes returns all valid authorization scopes defined in the system
func GetAllValidScopes() []string {
	return []string{
		ScopeMTLSRequestCert,
		ScopeMTLSReadCert,
		ScopeMTLSApproveCert,
		ScopeMTLSRenewCert,
		ScopeMTLSRevokeCert,
		ScopeMTLSDownloadCert,
		ScopeMTLSDownloadAllCerts,
		ScopeMTLSReadAllCerts,
		ScopeMTLSAutoApproveCert,
		ScopeMTLSSelfApproveCerts,
	}
}
