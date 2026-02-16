import { adminApi } from "./client";
import type {
  AgentCoreInfo,
  AgentCoreInstance,
  AgentCoreSwitchLog,
  AgentCoreSwitchLogsParams,
  AgentCoreSwitchLogsResponse,
  ConvertCoreConfigRequest,
  ConvertCoreConfigResult,
  CreateAgentCoreInstanceRequest,
  SwitchAgentCoreRequest,
  SwitchAgentCoreResult,
} from "@/types/admin";

type CoreInfoPayload = {
  type: string;
  version: string;
  installed: boolean;
  capabilities: string[];
};

type CoreInstancePayload = {
  ID: number;
  AgentHostID: number;
  InstanceID: string;
  CoreType: string;
  Status: string;
  ListenPorts: number[];
  ConfigTemplateID: number | null;
  ConfigHash: string;
  StartedAt: number | null;
  LastHeartbeatAt: number | null;
  ErrorMessage: string;
  CreatedAt: number;
  UpdatedAt: number;
};

type SwitchLogPayload = {
  ID: number;
  AgentHostID: number;
  FromInstanceID: string | null;
  FromCoreType: string | null;
  ToInstanceID: string;
  ToCoreType: string;
  OperatorID: number | null;
  Status: string;
  Detail: string;
  CreatedAt: number;
  CompletedAt: number | null;
};

type SwitchResultPayload = {
  success: boolean;
  new_instance_id?: string;
  message?: string;
  error?: string;
  switch_log_id?: number;
  completed_at?: number;
  from_instance_id?: string;
  to_core_type?: string;
};

const mapSwitchResult = (result: SwitchResultPayload): SwitchAgentCoreResult => ({
  success: result.success,
  new_instance_id: result.new_instance_id,
  message: result.message,
  error: result.error,
  switch_log_id: result.switch_log_id,
  completed_at: result.completed_at,
  from_instance_id: result.from_instance_id,
  to_core_type: result.to_core_type,
});

const mapCoreInfo = (core: CoreInfoPayload): AgentCoreInfo => ({
  type: core.type,
  version: core.version,
  installed: core.installed,
  capabilities: core.capabilities ?? [],
});

const mapCoreInstance = (instance: CoreInstancePayload): AgentCoreInstance => ({
  id: instance.ID,
  agent_host_id: instance.AgentHostID,
  instance_id: instance.InstanceID,
  core_type: instance.CoreType,
  status: instance.Status,
  listen_ports: instance.ListenPorts ?? [],
  config_template_id: instance.ConfigTemplateID ?? undefined,
  config_hash: instance.ConfigHash,
  started_at: instance.StartedAt ?? undefined,
  last_heartbeat_at: instance.LastHeartbeatAt ?? undefined,
  error_message: instance.ErrorMessage,
  created_at: instance.CreatedAt,
  updated_at: instance.UpdatedAt,
});

const mapSwitchLog = (log: SwitchLogPayload): AgentCoreSwitchLog => ({
  id: log.ID,
  agent_host_id: log.AgentHostID,
  from_instance_id: log.FromInstanceID ?? undefined,
  from_core_type: log.FromCoreType ?? undefined,
  to_instance_id: log.ToInstanceID,
  to_core_type: log.ToCoreType,
  operator_id: log.OperatorID ?? undefined,
  status: log.Status,
  detail: log.Detail,
  created_at: log.CreatedAt,
  completed_at: log.CompletedAt ?? undefined,
});

export async function listAgentCores(agentHostId: number): Promise<AgentCoreInfo[]> {
  const response = await adminApi.get<{ data: CoreInfoPayload[] }>(
    `/agent-hosts/${agentHostId}/cores`
  );
  return response.data.data.map(mapCoreInfo);
}

export async function listAgentCoreInstances(
  agentHostId: number
): Promise<AgentCoreInstance[]> {
  const response = await adminApi.get<{ data: CoreInstancePayload[] }>(
    `/agent-hosts/${agentHostId}/core-instances`
  );
  return response.data.data.map(mapCoreInstance);
}

export async function createAgentCoreInstance(
  agentHostId: number,
  payload: CreateAgentCoreInstanceRequest
): Promise<AgentCoreInstance> {
  const response = await adminApi.post<{ data: CoreInstancePayload }>(
    `/agent-hosts/${agentHostId}/core-instances`,
    payload
  );
  return mapCoreInstance(response.data.data);
}

export async function deleteAgentCoreInstance(
  agentHostId: number,
  instanceId: string
): Promise<void> {
  await adminApi.delete(`/agent-hosts/${agentHostId}/core-instances/${instanceId}`);
}

export async function switchAgentCore(
  agentHostId: number,
  payload: SwitchAgentCoreRequest
): Promise<SwitchAgentCoreResult> {
  const response = await adminApi.post<{ data: SwitchResultPayload }>(
    `/agent-hosts/${agentHostId}/core-switch`,
    payload
  );
  return mapSwitchResult(response.data.data);
}

export async function convertAgentCoreConfig(
  agentHostId: number,
  payload: ConvertCoreConfigRequest
): Promise<ConvertCoreConfigResult> {
  const response = await adminApi.post<{ data: ConvertCoreConfigResult }>(
    `/agent-hosts/${agentHostId}/core-convert`,
    payload
  );
  return response.data.data;
}

export async function listAgentCoreSwitchLogs(
  params: AgentCoreSwitchLogsParams
): Promise<AgentCoreSwitchLogsResponse> {
  const response = await adminApi.get<{ data: { logs: SwitchLogPayload[]; total: number } }>(
    `/agent-hosts/${params.agent_host_id}/core-switch-logs`,
    {
      params: {
        status: params.status,
        start_at: params.start_at,
        end_at: params.end_at,
        limit: params.limit,
        offset: params.offset,
      },
    }
  );
  return {
    logs: response.data.data.logs.map(mapSwitchLog),
    total: response.data.data.total,
  };
}
