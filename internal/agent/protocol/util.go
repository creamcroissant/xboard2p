package protocol

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// runCommand 执行带超时的 shell 命令。
func runCommand(ctx context.Context, command string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	name, args, err := splitCommand(command)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, string(output))
	}
	return nil
}

func splitCommand(command string) (string, []string, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", nil, fmt.Errorf("command is required")
	}

	parts := make([]string, 0, 4)
	var buf strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, r := range trimmed {
		switch {
		case escaped:
			buf.WriteRune(r)
			escaped = false
		case r == '\\' && !inSingle:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case !inSingle && !inDouble && (r == ' ' || r == '\t' || r == '\n'):
			if buf.Len() > 0 {
				parts = append(parts, buf.String())
				buf.Reset()
			}
		default:
			buf.WriteRune(r)
		}
	}

	if escaped || inSingle || inDouble {
		return "", nil, fmt.Errorf("invalid command: unclosed quote or escape")
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("command is required")
	}
	return parts[0], parts[1:], nil
}
