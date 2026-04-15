package devtools

const (
	PostgresPort = 5432
	RedisPort    = 6379
	AdminerPort  = 8090
	BackendPort  = 8080
	FrontendPort = 5173
	DebuggerPort = 2345

	PostgresHost = "127.0.0.1"
	RedisHost    = "127.0.0.1"

	ComposeFileInfra = "docker/compose.infra.yml"
	ComposeFileDev   = "docker/compose.dev.yml"
)
