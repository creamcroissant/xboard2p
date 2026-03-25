package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/creamcroissant/xboard/internal/agent/core"
	"github.com/creamcroissant/xboard/internal/agent/protocol"
	agentv1 "github.com/creamcroissant/xboard/pkg/pb/agent/v1"
)

func (a *Agent) syncCoreOperations(ctx context.Context) {
	if a.grpc == nil {
		return
	}
	resp, err := a.grpc.GetCoreOperations(ctx, []string{"pending", "claimed"}, 1)
	if err != nil {
		slog.Error("Failed to fetch core operations", "error", err)
		return
	}
	for _, operation := range resp.GetOperations() {
		if operation == nil {
			continue
		}
		statusValue, resultPayload, errMessage := a.executeCoreOperation(ctx, operation)
		if _, err := a.grpc.ReportCoreOperation(ctx, &agentv1.ReportCoreOperationRequest{OperationId: operation.GetId(), Status: statusValue, ResultPayload: resultPayload, ErrorMessage: errMessage}); err != nil {
			slog.Error("Failed to report core operation", "operation_id", operation.GetId(), "error", err)
		}
	}
}

func (a *Agent) executeCoreOperation(ctx context.Context, operation *agentv1.CoreOperation) (string, []byte, string) {
	if operation == nil {
		return "failed", nil, "empty core operation"
	}
	switch strings.TrimSpace(operation.GetOperationType()) {
	case "install":
		return a.executeInstallCore(ctx, operation)
	case "switch":
		return a.executeSwitchCore(ctx, operation)
	case "create":
		return a.executeCreateCoreInstance(ctx, operation)
	case "ensure":
		return a.executeEnsureCore(ctx, operation)
	default:
		return "failed", nil, fmt.Sprintf("unsupported core operation type %q", operation.GetOperationType())
	}
}

func (a *Agent) executeInstallCore(ctx context.Context, operation *agentv1.CoreOperation) (string, []byte, string) {
	var payload agentv1.InstallCorePayload
	if err := json.Unmarshal(operation.GetRequestPayload(), &payload); err != nil {
		return "failed", nil, err.Error()
	}
	installer := core.NewInstaller(core.InstallerConfig{ScriptPath: a.cfg.Core.InstallScriptPath, SingBoxBinaryPath: a.cfg.Core.SingBoxBinaryPath, XrayBinaryPath: a.cfg.Core.XrayBinaryPath, ServiceName: a.cfg.Protocol.ServiceName}, nil, nil)
	resp, err := installer.InstallCore(ctx, &agentv1.InstallCoreRequest{CoreType: operation.GetCoreType(), Action: payload.Action, Version: payload.Version, Channel: payload.Channel, Flavor: payload.Flavor, Activate: payload.Activate, RequestId: payload.RequestId})
	a.invalidateCapabilitiesCache()
	if err != nil {
		return "failed", nil, err.Error()
	}
	data, _ := json.Marshal(resp)
	if resp.GetRolledBack() {
		return "rolled_back", data, resp.GetError()
	}
	if !resp.GetSuccess() {
		return "failed", data, resp.GetError()
	}
	return "completed", data, ""
}

func (a *Agent) executeSwitchCore(ctx context.Context, operation *agentv1.CoreOperation) (string, []byte, string) {
	var payload agentv1.SwitchCorePayload
	if err := json.Unmarshal(operation.GetRequestPayload(), &payload); err != nil {
		return "failed", nil, err.Error()
	}
	configPath, err := a.resolveOperationConfigPath(payload.GetToCoreType(), payload.GetFromInstanceId(), "panel-core-switch")
	if err != nil {
		return "failed", nil, err.Error()
	}
	filename := strings.TrimSuffix(filepathBase(configPath), ".json") + ".json"
	if err := a.protoMgr.ApplyConfigWithCore(ctx, payload.GetToCoreType(), filename, payload.GetConfigJson()); err != nil {
		return "failed", nil, err.Error()
	}
	newInstanceID, err := a.coreMgr.SwitchCore(ctx, payload.GetFromInstanceId(), core.CoreType(payload.GetToCoreType()), configPath, int32SliceToInt(payload.GetListenPorts()))
	if err != nil {
		return "failed", nil, err.Error()
	}
	resultPayload, _ := json.Marshal(map[string]any{"new_instance_id": newInstanceID, "core_type": payload.GetToCoreType(), "config_path": configPath})
	return "completed", resultPayload, ""
}

func (a *Agent) executeCreateCoreInstance(ctx context.Context, operation *agentv1.CoreOperation) (string, []byte, string) {
	var payload agentv1.CreateCoreInstancePayload
	if err := json.Unmarshal(operation.GetRequestPayload(), &payload); err != nil {
		return "failed", nil, err.Error()
	}
	configPath, err := a.resolveOperationConfigPath(operation.GetCoreType(), payload.GetInstanceId(), "panel-core-create")
	if err != nil {
		return "failed", nil, err.Error()
	}
	filename := filepathBase(configPath)
	if err := a.protoMgr.ApplyConfigWithCore(ctx, operation.GetCoreType(), filename, payload.GetConfigJson()); err != nil {
		return "failed", nil, err.Error()
	}
	if err := a.coreMgr.StartInstance(ctx, core.CoreType(operation.GetCoreType()), payload.GetInstanceId(), configPath, nil); err != nil {
		return "failed", nil, err.Error()
	}
	resultPayload, _ := json.Marshal(map[string]any{"instance_id": payload.GetInstanceId(), "core_type": operation.GetCoreType(), "config_path": configPath})
	return "completed", resultPayload, ""
}

func (a *Agent) executeEnsureCore(ctx context.Context, operation *agentv1.CoreOperation) (string, []byte, string) {
	var payload agentv1.EnsureCorePayload
	if err := json.Unmarshal(operation.GetRequestPayload(), &payload); err != nil {
		return "failed", nil, err.Error()
	}
	installer := core.NewInstaller(core.InstallerConfig{ScriptPath: a.cfg.Core.InstallScriptPath, SingBoxBinaryPath: a.cfg.Core.SingBoxBinaryPath, XrayBinaryPath: a.cfg.Core.XrayBinaryPath, ServiceName: a.cfg.Protocol.ServiceName}, nil, nil)
	resp, err := installer.InstallCore(ctx, &agentv1.InstallCoreRequest{CoreType: operation.GetCoreType(), Action: "ensure", Version: payload.Version, Channel: payload.Channel, Flavor: payload.Flavor})
	a.invalidateCapabilitiesCache()
	if err != nil {
		return "failed", nil, err.Error()
	}
	data, _ := json.Marshal(resp)
	if !resp.GetSuccess() {
		return "failed", data, resp.GetError()
	}
	return "completed", data, ""
}

func (a *Agent) resolveOperationConfigPath(coreType, instanceID, fallbackPrefix string) (string, error) {
	trimmedCore := protocol.NormalizeCoreType(coreType)
	trimmedInstance := strings.TrimSpace(instanceID)
	if trimmedInstance == "" {
		trimmedInstance = fallbackPrefix
	}
	filename, err := protocol.SanitizeFilename(trimmedInstance + ".json")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(a.cfg.Protocol.ConfigDir)
	if dir == "" {
		dir = "/etc/sing-box/conf"
	}
	switch trimmedCore {
	case "xray":
		if strings.TrimSpace(a.cfg.Protocol.ManagedConfigDir) != "" {
			dir = a.cfg.Protocol.ManagedConfigDir
		}
	case "sing-box":
		if strings.TrimSpace(a.cfg.Protocol.ManagedConfigDir) != "" {
			dir = a.cfg.Protocol.ManagedConfigDir
		}
	}
	return dir + "/" + filename, nil
}

func filepathBase(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

func (a *Agent) invalidateCapabilitiesCache() {
	if a == nil {
		return
	}
	a.cachedCaps = nil
	a.capsDetectedAt = 0
}

func int32SliceToInt(input []int32) []int {
	if len(input) == 0 {
		return nil
	}
	result := make([]int, 0, len(input))
	for _, value := range input {
		result = append(result, int(value))
	}
	return result
}
