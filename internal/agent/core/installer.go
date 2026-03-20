package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creamcroissant/xboard/internal/agent/capability"
	"github.com/creamcroissant/xboard/internal/agent/initsys"
	"github.com/creamcroissant/xboard/internal/agent/protocol"
	agentv1 "github.com/creamcroissant/xboard/pkg/pb/agent/v1"
)

const installCommandTimeout = 15 * time.Minute

type InstallerConfig struct {
	ScriptPath        string
	SingBoxBinaryPath string
	XrayBinaryPath    string
	ServiceName       string
}

type Installer struct {
	cfg      InstallerConfig
	initSys  initsys.InitSystem
	detector *capability.Detector
	logger   *slog.Logger

	mu    sync.Mutex
	locks map[CoreType]*sync.Mutex
}

type installScriptResult struct {
	Success          bool   `json:"success"`
	CoreType         string `json:"core_type"`
	Action           string `json:"action"`
	RequestedRef     string `json:"requested_ref"`
	ResolvedTag      string `json:"resolved_tag"`
	Message          string `json:"message"`
	BinaryPath       string `json:"binary_path"`
	StableBinaryPath string `json:"stable_binary_path"`
	Changed          *bool  `json:"changed,omitempty"`
}

func NewInstaller(cfg InstallerConfig, initSys initsys.InitSystem, logger *slog.Logger) *Installer {
	if initSys == nil {
		initSys = initsys.Detect()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Installer{
		cfg:      cfg,
		initSys:  initSys,
		detector: capability.NewDetector(strings.TrimSpace(cfg.SingBoxBinaryPath), strings.TrimSpace(cfg.XrayBinaryPath)),
		logger:   logger,
		locks:    make(map[CoreType]*sync.Mutex),
	}
}

func (i *Installer) InstallCore(ctx context.Context, req *agentv1.InstallCoreRequest) (*agentv1.InstallCoreResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("empty install request")
	}

	coreType, err := normalizeInstallCoreType(req.CoreType)
	if err != nil {
		return nil, err
	}
	action, err := normalizeInstallAction(req.Action)
	if err != nil {
		return nil, err
	}
	version := strings.TrimSpace(req.Version)
	channel := strings.TrimSpace(req.Channel)
	flavor := strings.TrimSpace(req.Flavor)
	scriptPath := strings.TrimSpace(i.cfg.ScriptPath)
	if scriptPath == "" {
		return nil, fmt.Errorf("install script path is empty")
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("stat install script: %w", err)
	}

	lock := i.lockForCore(coreType)
	lock.Lock()
	defer lock.Unlock()

	previousVersion, previousErr := i.detectVersion(ctx, coreType)
	args := buildInstallCommandArgs(coreType, action, version, channel, flavor)
	scriptResult, err := i.runScript(ctx, scriptPath, args)
	if err != nil {
		return nil, err
	}

	currentVersion, currentErr := i.detectVersion(ctx, coreType)
	changed := installChanged(previousVersion, currentVersion, previousErr, currentErr, action, version)
	if err := currentErr; err != nil {
		return nil, fmt.Errorf("detect installed core version: %w", err)
	}

	resp := &agentv1.InstallCoreResponse{
		Success:         true,
		Changed:         changed,
		Message:         "core install completed",
		CoreType:        string(coreType),
		Version:         currentVersion,
		PreviousVersion: previousVersion,
	}
	applyScriptResult(resp, coreType, scriptResult)

	if req.Activate {
		activated, activateErr := i.activateCore(ctx, coreType)
		resp.Activated = activated
		if activateErr != nil {
			return i.handleActivationFailure(ctx, resp, coreType, previousVersion, previousErr, currentVersion, flavor, activateErr), nil
		}
	}

	return resp, nil
}

func (i *Installer) handleActivationFailure(ctx context.Context, resp *agentv1.InstallCoreResponse, coreType CoreType, previousVersion string, previousErr error, currentVersion, flavor string, activateErr error) *agentv1.InstallCoreResponse {
	resp.Success = false
	resp.Error = activateErr.Error()
	resp.Message = "core install completed but activation failed"
	resp.Activated = false
	i.logger.Error("failed to activate installed core", "core_type", string(coreType), "error", activateErr)

	if previousErr != nil || previousVersion == "" || currentVersion == "" || previousVersion == currentVersion {
		return resp
	}
	if err := i.rollbackCore(ctx, coreType, previousVersion, flavor); err != nil {
		resp.Error = fmt.Sprintf("%s; rollback failed: %v", activateErr.Error(), err)
		resp.Message = "core install activation failed and rollback failed"
		i.logger.Error("failed to roll back installed core", "core_type", string(coreType), "target_version", previousVersion, "error", err)
		return resp
	}
	if _, err := i.activateCore(ctx, coreType); err != nil {
		resp.Error = fmt.Sprintf("%s; rollback activation failed: %v", activateErr.Error(), err)
		resp.Message = "core install activation failed and rollback failed"
		i.logger.Error("failed to reactivate previous core version", "core_type", string(coreType), "target_version", previousVersion, "error", err)
		return resp
	}
	rolledBackVersion, err := i.detectVersion(ctx, coreType)
	if err != nil {
		resp.Error = fmt.Sprintf("%s; rollback verification failed: %v", activateErr.Error(), err)
		resp.Message = "core install activation failed and rollback verification failed"
		i.logger.Error("failed to verify rolled back core version", "core_type", string(coreType), "target_version", previousVersion, "error", err)
		return resp
	}

	resp.RolledBack = true
	resp.Version = rolledBackVersion
	resp.PreviousVersion = currentVersion
	resp.Message = "core install activation failed and previous version restored"
	return resp
}

func (i *Installer) lockForCore(coreType CoreType) *sync.Mutex {
	i.mu.Lock()
	defer i.mu.Unlock()
	if lock, ok := i.locks[coreType]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	i.locks[coreType] = lock
	return lock
}

func normalizeInstallCoreType(raw string) (CoreType, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case string(CoreTypeSingBox), "singbox":
		return CoreTypeSingBox, nil
	case string(CoreTypeXray):
		return CoreTypeXray, nil
	default:
		return "", fmt.Errorf("unsupported core_type %q", strings.TrimSpace(raw))
	}
}

func normalizeInstallAction(raw string) (string, error) {
	action := strings.TrimSpace(strings.ToLower(raw))
	switch action {
	case "install", "upgrade", "ensure":
		return action, nil
	default:
		return "", fmt.Errorf("unsupported action %q", strings.TrimSpace(raw))
	}
}

func buildInstallCommandArgs(coreType CoreType, action, version, channel, flavor string) []string {
	args := []string{
		"--core-action", action,
		"--core-type", string(coreType),
	}
	if version != "" {
		args = append(args, "--core-version", version)
	}
	if channel != "" {
		args = append(args, "--core-channel", channel)
	}
	if flavor != "" {
		args = append(args, "--core-flavor", flavor)
	}
	return args
}

func (i *Installer) runScript(ctx context.Context, scriptPath string, args []string) (*installScriptResult, error) {
	ctx, cancel := context.WithTimeout(ctx, installCommandTimeout)
	defer cancel()

	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.CommandContext(ctx, "/bin/sh", cmdArgs...)
	cmd.Env = append(os.Environ(), "XBOARD_AGENT_CORE_OUTPUT=json")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	i.logger.Info("running core install script", "script_path", scriptPath, "args", args)
	if err := cmd.Run(); err != nil {
		sanitized := protocol.SanitizeCommandOutput(output.Bytes())
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("core install command timed out: %s", sanitized)
		}
		if sanitized == "" {
			return nil, fmt.Errorf("core install command failed: %w", err)
		}
		return nil, fmt.Errorf("core install command failed: %w, output: %s", err, sanitized)
	}

	parsed, parseErr := parseInstallScriptOutput(output.Bytes())
	if parseErr == nil && parsed != nil {
		attrs := []any{
			"core_type", parsed.CoreType,
			"action", parsed.Action,
			"resolved_tag", parsed.ResolvedTag,
			"message", parsed.Message,
		}
		if parsed.Changed != nil {
			attrs = append(attrs, "changed", *parsed.Changed)
		}
		i.logger.Info("core install script completed", attrs...)
		return parsed, nil
	}

	if sanitized := protocol.SanitizeCommandOutput(output.Bytes()); sanitized != "" {
		i.logger.Info("core install script completed", "output", sanitized)
	}
	return nil, nil
}

func parseInstallScriptOutput(output []byte) (*installScriptResult, error) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, nil
	}
	lines := strings.Split(text, "\n")
	for idx := len(lines) - 1; idx >= 0; idx-- {
		candidate := strings.TrimSpace(lines[idx])
		if candidate == "" {
			continue
		}
		var result installScriptResult
		if err := json.Unmarshal([]byte(candidate), &result); err != nil {
			continue
		}
		if !result.Success {
			return nil, fmt.Errorf("install script reported unsuccessful result")
		}
		return &result, nil
	}
	return nil, fmt.Errorf("no structured install result found")
}

func applyScriptResult(resp *agentv1.InstallCoreResponse, coreType CoreType, result *installScriptResult) {
	if resp == nil || result == nil {
		return
	}
	if result.CoreType != "" && result.CoreType == string(coreType) {
		resp.CoreType = result.CoreType
	}
	if result.Message != "" {
		resp.Message = result.Message
	}
	if result.Changed != nil {
		resp.Changed = *result.Changed
	}
}

func (i *Installer) detectVersion(ctx context.Context, coreType CoreType) (string, error) {
	switch coreType {
	case CoreTypeSingBox:
		caps, err := i.detector.DetectSingBox(ctx)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(caps.CoreVersion), nil
	case CoreTypeXray:
		caps, err := i.detector.DetectXray(ctx)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(caps.CoreVersion), nil
	default:
		return "", fmt.Errorf("unsupported core type %q", coreType)
	}
}

func installChanged(previousVersion, currentVersion string, previousErr, currentErr error, action, requestedVersion string) bool {
	if currentErr != nil {
		return false
	}
	if previousErr != nil {
		return true
	}
	if currentVersion != previousVersion {
		return true
	}
	if requestedVersion != "" && requestedVersion != currentVersion {
		return true
	}
	return action == "install"
}

func (i *Installer) rollbackCore(ctx context.Context, coreType CoreType, version, flavor string) error {
	args := buildInstallCommandArgs(coreType, "ensure", strings.TrimSpace(version), "", strings.TrimSpace(flavor))
	if _, err := i.runScript(ctx, strings.TrimSpace(i.cfg.ScriptPath), args); err != nil {
		return fmt.Errorf("restore previous core version: %w", err)
	}
	return nil
}

func (i *Installer) activateCore(ctx context.Context, coreType CoreType) (bool, error) {
	service := strings.TrimSpace(i.cfg.ServiceName)
	if service == "" {
		switch coreType {
		case CoreTypeSingBox:
			service = "sing-box"
		case CoreTypeXray:
			service = "xray"
		}
	}
	if service == "" {
		return false, fmt.Errorf("service name is empty")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := i.initSys.Restart(ctx, service); err != nil {
		return false, fmt.Errorf("activate installed core: %w", err)
	}
	return true, nil
}
