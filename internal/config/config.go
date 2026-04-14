package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/Crowley723/conduit/internal/authorization"

	"gopkg.in/yaml.v3"
)

func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config file path is required (use -config or -c)")
	}

	// Read and parse YAML
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	applyEnvironmentOverrides(&config)

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

var (
	EnvOIDCClientID             = "CONDUIT_OIDC_CLIENT_ID"
	EnvOIDCClientSecret         = "CONDUIT_OIDC_CLIENT_SECRET"
	EnvOIDCIssuerURL            = "CONDUIT_OIDC_ISSUER_URL"
	EnvOIDCRedirectURL          = "CONDUIT_OIDC_REDIRECT_URL"
	EnvRedisPassword            = "CONDUIT_REDIS_PASSWORD"
	EnvRedisUsername            = "CONDUIT_REDIS_USERNAME"
	EnvRedisSentinelUsername    = "CONDUIT_REDIS_SENTINEL_USERNAME"
	EnvRedisSentinelPassword    = "CONDUIT_REDIS_SENTINEL_PASSWORD"
	EnvMTLSDownloadTokenHMACKey = "CONDUIT_MTLS_DOWNLOAD_TOKEN_HMAC_KEY"
	EnvStorageHost              = "CONDUIT_STORAGE_HOST"
	EnvStoragePort              = "CONDUIT_STORAGE_PORT"
	EnvStorageUsername          = "CONDUIT_STORAGE_USERNAME"
	EnvStoragePassword          = "CONDUIT_STORAGE_PASSWORD"
	EnvStorageDatabase          = "CONDUIT_STORAGE_DATABASE"
)

func applyEnvironmentOverrides(config *Config) {
	if clientID := os.Getenv(EnvOIDCClientID); clientID != "" {
		config.OIDC.ClientID = clientID
	}

	if clientSecret := os.Getenv(EnvOIDCClientSecret); clientSecret != "" {
		config.OIDC.ClientSecret = clientSecret
	}

	if issuerURL := os.Getenv(EnvOIDCIssuerURL); issuerURL != "" {
		config.OIDC.IssuerURL = issuerURL
	}

	if redirectURL := os.Getenv(EnvOIDCRedirectURL); redirectURL != "" {
		config.OIDC.RedirectURI = redirectURL
	}

	if redisPassword := os.Getenv(EnvRedisPassword); redisPassword != "" {
		if config.Redis == nil {
			config.Redis = &RedisConfig{}
		}
		config.Redis.Password = redisPassword
	}

	if redisUsername := os.Getenv(EnvRedisUsername); redisUsername != "" {
		if config.Redis == nil {
			config.Redis = &RedisConfig{}
		}
		config.Redis.Username = redisUsername
	}

	if sentinelUsername := os.Getenv(EnvRedisSentinelUsername); sentinelUsername != "" {
		if config.Redis == nil {
			config.Redis = &RedisConfig{}
		}
		if config.Redis.Sentinel == nil {
			config.Redis.Sentinel = &RedisSentinelConfig{}
		}
		config.Redis.Sentinel.SentinelUsername = sentinelUsername
	}

	if sentinelPassword := os.Getenv(EnvRedisSentinelPassword); sentinelPassword != "" {
		if config.Redis == nil {
			config.Redis = &RedisConfig{}
		}
		if config.Redis.Sentinel == nil {
			config.Redis.Sentinel = &RedisSentinelConfig{}
		}
		config.Redis.Sentinel.SentinelPassword = sentinelPassword
	}

	if hmacKey := os.Getenv(EnvMTLSDownloadTokenHMACKey); hmacKey != "" {
		config.MTLS.DownloadTokenHMACKey = hmacKey
	}

	if host := os.Getenv(EnvStorageHost); host != "" {
		if config.Storage == nil {
			config.Storage = &StorageConfig{}
		}
		config.Storage.Host = host
	}

	if portStr := os.Getenv(EnvStoragePort); portStr != "" {
		if config.Storage == nil {
			config.Storage = &StorageConfig{}
		}
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Storage.Port = port
		}
	}

	if username := os.Getenv(EnvStorageUsername); username != "" {
		if config.Storage == nil {
			config.Storage = &StorageConfig{}
		}
		config.Storage.Username = username
	}

	if password := os.Getenv(EnvStoragePassword); password != "" {
		if config.Storage == nil {
			config.Storage = &StorageConfig{}
		}
		config.Storage.Password = password
	}

	if database := os.Getenv(EnvStorageDatabase); database != "" {
		if config.Storage == nil {
			config.Storage = &StorageConfig{}
		}
		config.Storage.Database = database
	}
}

func validateConfig(config *Config) error {

	err := config.validateServerConfig()
	if err != nil {
		return err
	}

	err = config.validateOIDCConfig()
	if err != nil {
		return err
	}

	err = config.validateLogConfig()
	if err != nil {
		return err
	}

	err = config.validateCORSConfig()
	if err != nil {
		return err
	}

	err = config.validateSessionConfig()
	if err != nil {
		return err
	}

	if config.Sessions.Store == "redis" {
		err = config.validateRedisConfig()
		if err != nil {
			return err
		}
	}

	err = config.validateDistributedConfig()
	if err != nil {
		return err
	}

	err = config.validateStorageConfig()
	if err != nil {
		return err
	}

	err = config.validateAuthorizationConfig()
	if err != nil {
		return err
	}

	err = config.validateMTLSConfig()
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) validateOIDCConfig() error {
	if c.OIDC.ClientID == "" {
		return fmt.Errorf("oidc client id is required")
	}

	if c.OIDC.ClientSecret == "" {
		return fmt.Errorf("OIDC clientSecret is required")
	}

	if err := validateURL(c.OIDC.IssuerURL, "issuer_url"); err != nil {
		return err
	}

	if err := validateURL(c.OIDC.RedirectURI, "redirect_url"); err != nil {
		return err
	}

	if len(c.OIDC.Scopes) == 0 {
		c.OIDC.Scopes = DefaultOIDCConfig.Scopes
	}

	return nil
}

func (c *Config) validateServerConfig() error {
	if c.Server.Port == 0 {
		c.Server.Port = DefaultServerConfig.Port
	}

	if c.Server.ExternalURL == "" {
		return fmt.Errorf("server.external_url is required")
	}

	if c.Server.Debug != nil && c.Server.Debug.Enabled {
		if c.Server.Debug.Host == "" {
			c.Server.Debug.Host = DefaultDebugConfig.Host
		}
		if c.Server.Debug.Port <= 0 || c.Server.Debug.Port >= 65535 {
			c.Server.Debug.Port = DefaultDebugConfig.Port
		}
	}

	return nil
}

func (c *Config) validateLogConfig() error {
	if c.Log.Format == "" {
		c.Log.Format = DefaultLogConfig.Format
	} else {
		switch c.Log.Format {
		case "text":
			c.Log.Format = "text"
		case "json":
			c.Log.Format = "json"
		default:
			return fmt.Errorf("invalid log format: %s, options are text or json", c.Log.Format)
		}
	}

	if c.Log.Level == "" {
		c.Log.Level = DefaultLogConfig.Level
	} else {
		switch c.Log.Level {
		case "debug":
			c.Log.Level = string(rune(slog.LevelDebug))
		case "info":
			c.Log.Level = string(rune(slog.LevelInfo))
		case "warn":
			c.Log.Level = string(rune(slog.LevelWarn))
		case "error":
			c.Log.Level = string(rune(slog.LevelError))
		default:
			return fmt.Errorf("invalid log level: %s, options are debug, info, warn, error", c.Log.Level)
		}
	}

	return nil
}

func (c *Config) validateCORSConfig() error {
	if len(c.CORS.AllowedOrigins) == 0 {
		c.CORS.AllowedOrigins = DefaultCORSConfig.AllowedOrigins
	}
	if len(c.CORS.AllowedMethods) == 0 {
		c.CORS.AllowedMethods = DefaultCORSConfig.AllowedMethods
	}
	if len(c.CORS.AllowedHeaders) == 0 {
		c.CORS.AllowedHeaders = DefaultCORSConfig.AllowedHeaders
	}
	if c.CORS.MaxAgeSeconds == 0 {
		c.CORS.MaxAgeSeconds = DefaultCORSConfig.MaxAgeSeconds
	}

	return nil
}

func (c *Config) validateSessionConfig() error {
	if c == nil {
		return fmt.Errorf("session config is required")
	}

	if c.Sessions.Store == "" {
		c.Sessions.Store = DefaultSessionConfig.Store
	} else {
		switch c.Sessions.Store {
		case "memory":
			c.Sessions.Store = "memory"
		case "redis":
			c.Sessions.Store = "redis"
		default:
			return fmt.Errorf("invalid session store: %s, options are 'memory' or 'redis'", c.Sessions.Store)
		}
	}

	if c.Sessions.DurationSource == "" {
		c.Sessions.DurationSource = DefaultSessionConfig.DurationSource
	} else {
		switch c.Sessions.DurationSource {
		case "fixed":
			c.Sessions.DurationSource = "fixed"
		case "oidc_tokens":
			c.Sessions.DurationSource = "oidc_tokens"
		default:
			return fmt.Errorf("invalid session duration source: %s, options are 'fixed' or 'oidc_tokens'", c.Sessions.DurationSource)
		}
	}

	if c.Sessions.Name == "" {
		c.Sessions.Name = DefaultSessionConfig.Name
	}

	if c.Sessions.FixedTimeout == 0 {
		c.Sessions.FixedTimeout = DefaultSessionConfig.FixedTimeout
	}

	return nil
}

func (c *Config) validateRedisConfig() error {
	if c.Redis == nil {
		return fmt.Errorf("redis config is nil")
	}

	if c.Redis.Address == "" {
		return fmt.Errorf("redis address is required")
	}

	if _, _, err := net.SplitHostPort(c.Redis.Address); err != nil {
		return fmt.Errorf("invalid redis address format (expected host:port): %w", err)
	}

	// Apply default indices if not set
	if c.Redis.SessionIndex == 0 && c.Redis.LeaderIndex == 0 {
		c.Redis.SessionIndex = DefaultRedisConfig.SessionIndex
		c.Redis.LeaderIndex = DefaultRedisConfig.LeaderIndex
	}

	if c.Redis.SessionIndex < 0 {
		return fmt.Errorf("redis session_index must be non-negative, got %d", c.Redis.SessionIndex)
	}

	if c.Redis.LeaderIndex < 0 {
		return fmt.Errorf("redis leader_index must be non-negative, got %d", c.Redis.LeaderIndex)
	}

	if c.Redis.LeaderIndex == c.Redis.SessionIndex {
		return fmt.Errorf("redis leader_index and session_index should be different to avoid data collision (both are %d)", c.Redis.LeaderIndex)
	}

	const maxRedisDB = 15
	if c.Redis.SessionIndex > maxRedisDB {
		return fmt.Errorf("redis session_index %d exceeds typical maximum of %d", c.Redis.SessionIndex, maxRedisDB)
	}

	if c.Redis.LeaderIndex > maxRedisDB {
		return fmt.Errorf("redis leader_index %d exceeds typical maximum of %d", c.Redis.LeaderIndex, maxRedisDB)
	}

	if c.Redis.Sentinel != nil {
		if c.Redis.Sentinel.MasterName == "" {
			return fmt.Errorf("sentinel master_name is required")
		}
		if len(c.Redis.Sentinel.SentinelAddresses) == 0 {
			return fmt.Errorf("at least one sentinel address is required")
		}
	}
	return nil
}

func (c *Config) validateDistributedConfig() error {
	if c.Distributed == nil {
		return nil
	}

	// Apply default enabled state if not explicitly set
	if !c.Distributed.Enabled {
		return nil
	}

	if c.Distributed.TTL.Seconds() <= 0 {
		c.Distributed.TTL = DefaultDistributedConfig.TTL
	} else if c.Distributed.TTL > time.Minute {
		return fmt.Errorf("distributed ttl cannot be more than 1 minute")
	}

	return nil
}

func (c *Config) validateStorageConfig() error {
	if c.Storage == nil {
		return nil
	}

	if c.Storage.Host == "" {
		return fmt.Errorf("storage.host is required")
	}

	if c.Storage.Port <= 0 || c.Storage.Port > 65535 {
		return fmt.Errorf("storage.port must be between 1 and 65535, got %d", c.Storage.Port)
	}

	if c.Storage.Database == "" {
		return fmt.Errorf("storage.database is required")
	}

	return nil
}

func (c *Config) validateMTLSConfig() error {
	if !c.MTLS.Enabled {
		return nil
	}

	if c.Storage == nil {
		return fmt.Errorf("storage is required when mtls is enabled")
	}

	if c.MTLS.DownloadTokenHMACKey == "" {
		return fmt.Errorf("mtls.download_token_hmac_key is required when mtls is enabled")
	}

	if len(c.MTLS.DownloadTokenHMACKey) <= 32 {
		return fmt.Errorf("mtls.download_token_hmac_key must be at least 32 characters")
	}

	if c.MTLS.MinCertificateValidityDays == 0 {
		c.MTLS.MinCertificateValidityDays = DefaultMTLSIssuerConfig.MinCertificateValidityDays
	}

	if c.MTLS.MaxCertificateValidityDays == 0 {
		c.MTLS.MaxCertificateValidityDays = DefaultMTLSIssuerConfig.MaxCertificateValidityDays
	}

	if c.MTLS.MaxCertificateValidityDays < c.MTLS.MinCertificateValidityDays {
		return fmt.Errorf("mtls.max_certificate_validity_days cannot be less than min_certificate_validity_days")
	}

	if c.MTLS.Kubernetes == nil {
		c.MTLS.Kubernetes = DefaultMTLSManagementKubernetesConfig
	}

	if c.MTLS.CertificateSubject == nil {
		c.MTLS.CertificateSubject = DefaultCertificateSubject
	}

	if c.MTLS.CertificateSubject.Organization == "" {
		c.MTLS.CertificateSubject.Organization = DefaultCertificateSubject.Organization
	}

	if c.MTLS.BackgroundJobConfig == nil {
		c.MTLS.BackgroundJobConfig = DefaultMTLSBackgroundJobConfig
	}

	if c.MTLS.BackgroundJobConfig.ApprovedCertificatePollingInterval == 0 {
		c.MTLS.BackgroundJobConfig.ApprovedCertificatePollingInterval = DefaultMTLSBackgroundJobConfig.ApprovedCertificatePollingInterval
	}

	if c.MTLS.BackgroundJobConfig.IssuedCertificatePollingInterval == 0 {
		c.MTLS.BackgroundJobConfig.IssuedCertificatePollingInterval = DefaultMTLSBackgroundJobConfig.IssuedCertificatePollingInterval
	}

	return c.validateMTLSKubernetesConfig()
}

func (c *Config) validateMTLSKubernetesConfig() error {
	if c.MTLS.Kubernetes == nil || !c.MTLS.Kubernetes.Enabled {
		return nil
	}

	if c.MTLS.Kubernetes.Issuer == nil {
		return fmt.Errorf("mtls.kubernetes.issuer is required when mtls.kubernetes is enabled")
	}

	if c.MTLS.Kubernetes.Issuer.Name == "" {
		return fmt.Errorf("mtls.kubernetes.issuer.name is required when mtls.kubernetes is enabled")
	}

	if c.MTLS.Kubernetes.Issuer.Kind == "" {
		return fmt.Errorf("mtls.kubernetes.issuer.kind is required when mtls.kubernetes is enabled")
	}

	kind := c.MTLS.Kubernetes.Issuer.Kind
	if kind != "Issuer" && kind != "ClusterIssuer" {
		return fmt.Errorf("mtls.kubernetes.issuer.kind must be either 'Issuer' or 'ClusterIssuer', got '%s'", kind)
	}

	return nil
}

func (c *Config) validateAuthorizationConfig() error {
	if c.Authorization.GroupScopes == nil || len(c.Authorization.GroupScopes) == 0 {
		c.Authorization = DefaultAuthorizationConfig
	}

	validScopes := authorization.GetAllValidScopes()
	for group, scopes := range c.Authorization.GroupScopes {
		if len(scopes) == 0 {
			return fmt.Errorf("authorization group '%s' has no scopes defined", group)
		}

		for _, scope := range scopes {
			if !slices.Contains(validScopes, scope) {
				return fmt.Errorf("authorization group '%s' contains invalid scope '%s'", group, scope)
			}
		}
	}

	return nil
}
