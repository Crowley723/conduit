package devtools

import (
	"context"
	"fmt"
	"time"
)

// UpCommand starts the development infrastructure
type UpCommand struct {
	Mode string `help:"Development mode" enum:"default,k8s" default:"default"`
}

func (c *UpCommand) Run() error {
	fmt.Println("Starting development infrastructure...")

	if c.Mode == "k8s" {
		return fmt.Errorf("k8s mode not yet implemented")
	}

	fmt.Print("  Starting containers... ")
	if err := ComposeUp(ComposeFileInfra); err != nil {
		fmt.Println("failed")
		return err
	}
	fmt.Println("done")

	ctx := context.Background()

	fmt.Print("  Waiting for PostgreSQL... ")
	if err := WaitForPostgres(ctx, PostgresHost, PostgresPort, 30*time.Second); err != nil {
		fmt.Println("failed")
		return err
	}
	fmt.Printf("ready (%s:%d)\n", PostgresHost, PostgresPort)

	fmt.Print("  Waiting for Redis... ")
	if err := WaitForRedis(ctx, RedisHost, RedisPort, 30*time.Second); err != nil {
		fmt.Println("failed")
		return err
	}
	fmt.Printf("ready (%s:%d)\n", RedisHost, RedisPort)

	fmt.Printf("  Adminer UI: http://%s:%d\n", PostgresHost, AdminerPort)

	fmt.Println("\nInfrastructure is ready!")
	fmt.Println("Run 'conduit-dev serve' to start the application")
	return nil
}

// DownCommand stops the development infrastructure
type DownCommand struct {
	All bool `help:"Stop both infrastructure and app containers" default:"false"`
}

func (c *DownCommand) Run() error {
	fmt.Println("Stopping development environment...")

	if c.All {
		if err := ComposeDown(ComposeFileDev); err != nil {
			return err
		}
	} else {
		if err := ComposeDown(ComposeFileInfra); err != nil {
			return err
		}
	}

	fmt.Println("Stopped")
	return nil
}

// StatusCommand shows the status of development services
type StatusCommand struct{}

func (c *StatusCommand) Run() error {
	fmt.Println("Development Environment Status:\n")

	infraStatuses, err := ComposeStatus(ComposeFileInfra)
	if err != nil {
		return err
	}

	devStatuses, err := ComposeStatus(ComposeFileDev)
	if err != nil {
		return err
	}

	statuses := append(infraStatuses, devStatuses...)

	if len(statuses) == 0 {
		fmt.Println("  No services running")
		fmt.Println("\nRun 'conduit-dev up' to start infrastructure")
		return nil
	}

	for _, status := range statuses {
		statusIcon := "✗"
		if status.Status == "running" {
			statusIcon = "✓"
		}

		healthInfo := ""
		if status.Health != "" {
			healthInfo = fmt.Sprintf(" [%s]", status.Health)
		}

		fmt.Printf("  %s %s: %s%s\n", statusIcon, status.Name, status.Status, healthInfo)
	}

	ctx := context.Background()

	fmt.Println("\nConnectivity:")

	if err := CheckPostgres(PostgresHost, PostgresPort); err == nil {
		fmt.Printf("  ✓ PostgreSQL: accessible at %s:%d\n", PostgresHost, PostgresPort)
	} else {
		fmt.Printf("  ✗ PostgreSQL: %v\n", err)
	}

	if err := CheckRedis(ctx, RedisHost, RedisPort); err == nil {
		fmt.Printf("  ✓ Redis: accessible at %s:%d\n", RedisHost, RedisPort)
	} else {
		fmt.Printf("  ✗ Redis: %v\n", err)
	}

	return nil
}

// LogsCommand tails service logs
type LogsCommand struct {
	Service string `arg:"" optional:"" help:"Service name (postgres, redis, adminer, app) - omit for all"`
	App     bool   `help:"Show app container logs" default:"false"`
}

func (c *LogsCommand) Run() error {
	composeFile := ComposeFileInfra
	if c.App {
		composeFile = ComposeFileDev
	}
	return ComposeLogs(composeFile, c.Service)
}

// ServeCommand starts the backend and frontend with hot-reload in containers
type ServeCommand struct {
	Mode string `help:"Development mode" enum:"default,k8s" default:"default"`
}

func (c *ServeCommand) Run() error {
	if c.Mode == "k8s" {
		return fmt.Errorf("k8s mode not yet implemented")
	}

	fmt.Println("Starting full development environment (containerized)...")
	fmt.Println()
	fmt.Println("Services:")
	fmt.Printf("  Backend:  http://localhost:%d\n", BackendPort)
	fmt.Printf("  Frontend: http://localhost:%d\n", FrontendPort)
	fmt.Printf("  Adminer:  http://localhost:%d\n", AdminerPort)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	return ComposeUpAttached(ComposeFileDev)
}
