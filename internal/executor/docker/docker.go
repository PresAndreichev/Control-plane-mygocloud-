package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Executor struct{}

func New() (*Executor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "docker", "version").Run(); err != nil {
		return nil, fmt.Errorf("docker not available: %w", err)
	}
	return &Executor{}, nil
}

func (e *Executor) Deploy(ctx context.Context, appID, depID, imageRef, version string) (string, error) {
	fullImage := imageRef + ":" + version

	// Pull with 2-minute timeout
	pullCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	pull := exec.CommandContext(pullCtx, "docker", "pull", fullImage)
	if out, err := pull.CombinedOutput(); err != nil {
		return "", fmt.Errorf("docker pull %s: %w\n%s", fullImage, err, string(out))
	}

	// Run with 30-second timeout
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	name := fmt.Sprintf("mygocloud-%s-%s", appID[:8], depID[:8])
	run := exec.CommandContext(runCtx,
		"docker", "run", "-d",
		"--name", name,
		"--label", "mygocloud.app.id="+appID,
		"--label", "mygocloud.deployment.id="+depID,
		"--label", "mygocloud.version="+version,
		"--label", "mygocloud.managed=true",
		fullImage,
	)
	out, err := run.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run: %w\n%s", err, string(out))
	}

	containerID := strings.TrimSpace(string(out))
	return containerID, nil
}

func (e *Executor) Status(ctx context.Context, containerID string) (string, error) {
	// 5-second timeout — inspect should be instant
	inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(inspectCtx,
		"docker", "inspect", "--format",
		"{{.State.Status}}|{{.State.ExitCode}}",
		containerID,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No such object") || strings.Contains(err.Error(), "No such") {
			return "failed", nil
		}
		return "", fmt.Errorf("docker inspect: %w\n%s", err, string(out))
	}

	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	state := parts[0]

	switch state {
	case "running":
		return "running", nil
	case "exited":
		if len(parts) > 1 && parts[1] == "0" {
			return "successful", nil
		}
		return "failed", nil
	case "dead", "removing":
		return "failed", nil
	default:
		return "pending", nil
	}
}

func (e *Executor) Logs(ctx context.Context, containerID string, tail int) (string, error) {
	cmd := exec.CommandContext(ctx,
		"docker", "logs", "--tail", strconv.Itoa(tail),
		containerID,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs: %w\n%s", err, string(out))
	}
	return string(out), nil
}

func (e *Executor) Stop(ctx context.Context, containerID string) error {
	stopCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_ = exec.CommandContext(stopCtx, "docker", "stop", "-t", "10", containerID).Run()

	rmCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	rm := exec.CommandContext(rmCtx, "docker", "rm", "-f", containerID)
	if out, err := rm.CombinedOutput(); err != nil {
		return fmt.Errorf("docker rm: %w\n%s", err, string(out))
	}
	return nil
}
