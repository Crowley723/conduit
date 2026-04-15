package devtools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ComposeUp starts the infrastructure containers
func ComposeUp(composeFile string) error {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start containers: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// ComposeDown stops and removes containers
func ComposeDown(composeFile string) error {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "down")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop containers: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// ComposeUpAttached starts containers and attaches to logs
func ComposeUpAttached(composeFile string) error {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "up", "--build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name   string
	Status string
	Health string
}

// ComposeStatus returns the status of all services
func ComposeStatus(composeFile string) ([]ServiceStatus, error) {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "ps", "--format", "json")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		// If compose file doesn't exist or no containers, return empty list
		return []ServiceStatus{}, nil
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var statuses []ServiceStatus

	for _, line := range lines {
		if line == "" {
			continue
		}

		name := extractJSONField(line, "Service")
		status := extractJSONField(line, "State")
		health := extractJSONField(line, "Health")

		if name != "" {
			statuses = append(statuses, ServiceStatus{
				Name:   name,
				Status: status,
				Health: health,
			})
		}
	}

	return statuses, nil
}

// ComposeLogs tails logs for a specific service or all services
func ComposeLogs(composeFile, service string) error {
	args := []string{"compose", "-f", composeFile, "logs", "-f"}
	if service != "" {
		args = append(args, service)
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// Helper function to extract a field from JSON string
func extractJSONField(jsonStr, field string) string {
	pattern := fmt.Sprintf(`"%s":"`, field)
	start := strings.Index(jsonStr, pattern)
	if start == -1 {
		return ""
	}
	start += len(pattern)

	end := strings.Index(jsonStr[start:], `"`)
	if end == -1 {
		return ""
	}

	return jsonStr[start : start+end]
}
