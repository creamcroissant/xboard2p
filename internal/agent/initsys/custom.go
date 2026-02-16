package initsys

import (
	"context"
    "fmt"
	"strings"
)

// Custom implements InitSystem using user-defined shell commands.
type Custom struct {
	commands CustomCommands
}

func (c *Custom) Type() string {
	return "custom"
}

func (c *Custom) Start(ctx context.Context, service string) error {
	cmd := strings.ReplaceAll(c.commands.Start, "{{service}}", service)
    // Also try {service} for backward compatibility
    cmd = strings.ReplaceAll(cmd, "{service}", service)
	fmt.Printf("DEBUG: Custom.Start service=%s cmd=%s\n", service, cmd)
	return runCommand(ctx, cmd)
}

func (c *Custom) Stop(ctx context.Context, service string) error {
	cmd := strings.ReplaceAll(c.commands.Stop, "{{service}}", service)
    cmd = strings.ReplaceAll(cmd, "{service}", service)
	fmt.Printf("DEBUG: Custom.Stop service=%s cmd=%s\n", service, cmd)
	err := runCommand(ctx, cmd)
    if err != nil {
        // Fallback: try pkill if defined command failed
        fmt.Printf("DEBUG: Custom.Stop failed, trying pkill fallback for service=%s\n", service)
        // Try to kill by pid file first if it exists (assuming standard location pattern from our tests)
        // But since we can't easily know the pid file path here without parsing cmd, we'll try pkill -f
        // Note: This is a rough fallback for development/testing environments
        pkillCmd := fmt.Sprintf("pkill -9 -f '%s.json'", service)
        _ = runCommand(ctx, pkillCmd)
    }
    return err
}

func (c *Custom) Restart(ctx context.Context, service string) error {
	if c.commands.Restart != "" {
		cmd := strings.ReplaceAll(c.commands.Restart, "{{service}}", service)
		return runCommand(ctx, cmd)
	}
	// Fallback: stop then start
	if err := c.Stop(ctx, service); err != nil {
		// Ignore stop errors
	}
	return c.Start(ctx, service)
}

func (c *Custom) Reload(ctx context.Context, service string) error {
	if c.commands.Reload != "" {
		cmd := strings.ReplaceAll(c.commands.Reload, "{{service}}", service)
		return runCommand(ctx, cmd)
	}
	// Fallback to restart
	return c.Restart(ctx, service)
}

func (c *Custom) Status(ctx context.Context, service string) (bool, error) {
	if c.commands.Status == "" {
		return false, nil
	}
	cmd := strings.ReplaceAll(c.commands.Status, "{{service}}", service)
	_, err := runCommandWithOutput(ctx, cmd)
	// Status command returns 0 if running
	return err == nil, nil
}

func (c *Custom) Enable(ctx context.Context, service string) error {
	if c.commands.Enable == "" {
		return nil // Not supported
	}
	cmd := strings.ReplaceAll(c.commands.Enable, "{{service}}", service)
	return runCommand(ctx, cmd)
}

func (c *Custom) Disable(ctx context.Context, service string) error {
	if c.commands.Disable == "" {
		return nil // Not supported
	}
	cmd := strings.ReplaceAll(c.commands.Disable, "{{service}}", service)
	return runCommand(ctx, cmd)
}
