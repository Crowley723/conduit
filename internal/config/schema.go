package config

import (
	"github.com/Crowley723/conduit/internal/authorization"
	"time"
)

type Config struct {
	Server        ServerConfig        `yaml:"server"`
	OIDC          OIDCConfig          `yaml:"oidc"`
	Log           LogConfig           `yaml:"log"`
	CORS          CORSConfig          `yaml:"cors"`
	Sessions      SessionConfig       `yaml:"sessions"`
	Authorization AuthorizationConfig `yaml:"authorization"`
	Redis         *RedisConfig        `yaml:"redis"`
	Distributed   *DistributedConfig  `yaml:"distributed"`
	Storage       *StorageConfig      `yaml:"storage"`
	Features      *FeaturesConfig     `yaml:"features"`
}

type ServerConfig struct {
	Port        int                `yaml:"port"`
	ExternalURL string             `yaml:"external_url"`
	Debug       *ServerDebugConfig `yaml:"debug"`
}

var DefaultServerConfig = ServerConfig{
	Port:        8080,
	ExternalURL: "localhost:8080",
}

type ServerDebugConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
}

var DefaultDebugConfig = ServerDebugConfig{
	Enabled: false,
	Host:    "localhost",
	Port:    5123,
}

type OIDCConfig struct {
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	IssuerURL    string   `yaml:"issuer_url"`
	RedirectURI  string   `yaml:"redirect_url"`
	Scopes       []string `yaml:"scopes"`
}

var DefaultOIDCConfig = OIDCConfig{
	Scopes: []string{"openid", "profile", "email", "groups"},
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

var DefaultLogConfig = LogConfig{
	Level:  "info",
	Format: "text",
}

type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	ExposedHeaders   []string `yaml:"exposed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAgeSeconds    int      `yaml:"max_age_seconds"`
}

var DefaultCORSConfig = CORSConfig{
	AllowedOrigins: []string{"http://localhost:5173"},
	AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	AllowedHeaders: []string{"*"},
	MaxAgeSeconds:  300,
}

type SessionConfig struct {
	Store          string        `yaml:"store"`
	DurationSource string        `yaml:"duration_source"`
	FixedTimeout   time.Duration `yaml:"fixed_timeout"`
	Name           string        `yaml:"name"`
	Secure         bool          `yaml:"secure"`
}

var DefaultSessionConfig = SessionConfig{
	Store:          "memory",
	DurationSource: "fixed",
	FixedTimeout:   24 * time.Hour,
	Name:           "session_id",
	Secure:         true,
}

type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type RedisConfig struct {
	Address      string               `yaml:"address"`
	Username     string               `yaml:"username"`
	Password     string               `yaml:"password"`
	Sentinel     *RedisSentinelConfig `yaml:"sentinel"`
	SessionIndex int                  `yaml:"session_index"`
	CacheIndex   int                  `yaml:"cache_index"`
	LeaderIndex  int                  `yaml:"leader_index"`
}

var DefaultRedisConfig = RedisConfig{
	SessionIndex: 0,
	CacheIndex:   1,
	LeaderIndex:  2,
}

type RedisSentinelConfig struct {
	MasterName        string   `yaml:"master_name"`
	SentinelAddresses []string `yaml:"addresses"`
	SentinelPassword  string   `yaml:"password"`
	SentinelUsername  string   `yaml:"username"`
}

type DistributedConfig struct {
	Enabled bool          `yaml:"enabled"`
	TTL     time.Duration `yaml:"ttl"`
}

var DefaultDistributedConfig = DistributedConfig{
	Enabled: false,
	TTL:     30 * time.Second,
}

type StorageConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

var DefaultStorageConfig = StorageConfig{
	Enabled: false,
}

type FeaturesConfig struct {
	MTLSManagement MTLSManagement `yaml:"mtls_management,omitempty"`
}

var DefaultFeaturesConfig = FeaturesConfig{
	MTLSManagement: DefaultMTLSIssuerConfig,
}

type MTLSManagement struct {
	Enabled                         bool                     `yaml:"enabled"`
	DownloadTokenHMACKey            string                   `yaml:"download_token_hmac_key"`
	AutoApproveAdminRequests        bool                     `yaml:"auto_approve_admin_requests"`
	AllowAdminsToApproveOwnRequests bool                     `yaml:"allow_admins_to_approve_own_requests"`
	MinCertificateValidityDays      int                      `yaml:"min_certificate_validity_days"`
	MaxCertificateValidityDays      int                      `yaml:"max_certificate_validity_days"`
	CertificateSubject              *CertificateSubject      `yaml:"certificate_subject,omitempty"`
	BackgroundJobConfig             *MTLSBackgroundJobConfig `yaml:"background_job_config,omitempty"`
	Kubernetes                      *KubernetesConfig        `yaml:"kubernetes,omitempty"`
}

type KubernetesConfig struct {
	Enabled    bool               `yaml:"enabled"`
	Namespace  string             `yaml:"namespace"`
	Kubeconfig string             `yaml:"kubeconfig"`
	InCluster  bool               `yaml:"in_cluster"`
	Issuer     *CertificateIssuer `yaml:"issuer"`
}

var DefaultMTLSManagementKubernetesConfig = &KubernetesConfig{
	Enabled:   false,
	InCluster: true,
	Namespace: "conduit",
	Issuer:    nil,
}

type CertificateIssuer struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
}

type CertificateSubject struct {
	Organization string `yaml:"organization"`
	Country      string `yaml:"country"`
	Locality     string `yaml:"locality"`
	Province     string `yaml:"province"`
}

var DefaultCertificateSubject = &CertificateSubject{
	Organization: "Homelab Conduit",
	Country:      "",
	Locality:     "",
	Province:     "",
}

type MTLSBackgroundJobConfig struct {
	ApprovedCertificatePollingInterval time.Duration `yaml:"approved_certificate_polling_interval"`
	IssuedCertificatePollingInterval   time.Duration `yaml:"issued_certificate_polling_interval"`
}

var DefaultMTLSBackgroundJobConfig = &MTLSBackgroundJobConfig{
	ApprovedCertificatePollingInterval: 30 * time.Second,
	IssuedCertificatePollingInterval:   30 * time.Second,
}

var DefaultMTLSIssuerConfig = MTLSManagement{
	Enabled:                         false,
	AutoApproveAdminRequests:        false,
	AllowAdminsToApproveOwnRequests: true,
	MinCertificateValidityDays:      30,
	MaxCertificateValidityDays:      365,
	Kubernetes:                      DefaultMTLSManagementKubernetesConfig,
	CertificateSubject:              DefaultCertificateSubject,
	BackgroundJobConfig:             DefaultMTLSBackgroundJobConfig,
}

type AuthorizationConfig struct {
	GroupScopes map[string][]string `yaml:"group_scopes"`
}

var DefaultAuthorizationConfig = AuthorizationConfig{
	GroupScopes: map[string][]string{
		"conduit:mtls:admin": {
			authorization.ScopeMTLSRequestCert,
			authorization.ScopeMTLSReadAllCerts,
			authorization.ScopeMTLSReadCert,
			authorization.ScopeMTLSApproveCert,
			authorization.ScopeMTLSRenewCert,
			authorization.ScopeMTLSRevokeCert,
			authorization.ScopeMTLSDownloadAllCerts,
			authorization.ScopeMTLSDownloadCert,
			authorization.ScopeMTLSAutoApproveCert,
		},
		"conduit:mtls:user": {
			authorization.ScopeMTLSRequestCert,
			authorization.ScopeMTLSReadCert,
			authorization.ScopeMTLSRenewCert,
			authorization.ScopeMTLSDownloadCert,
		},
	},
}
