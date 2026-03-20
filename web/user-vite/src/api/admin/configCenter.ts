import { adminApi } from "./client";
import type {
  ConfigCenterAppliedSnapshot,
  ConfigCenterApplyRun,
  ConfigCenterApplyRunDetail,
  ConfigCenterApplyRunListResponse,
  ConfigCenterArtifactListResponse,
  ConfigCenterDriftStateListResponse,
  ConfigCenterSemanticDiff,
  ConfigCenterSpec,
  ConfigCenterSpecHistoryResponse,
  ConfigCenterSpecListResponse,
  ConfigCenterTextDiff,
  CreateConfigCenterApplyRunRequest,
  GetConfigCenterApplyRunDetailParams,
  GetConfigCenterSemanticDiffParams,
  GetConfigCenterTextDiffParams,
  ImportConfigCenterSpecRequest,
  ImportConfigCenterSpecResult,
  ListConfigCenterAppliedSnapshotParams,
  ListConfigCenterApplyRunsParams,
  ListConfigCenterArtifactsParams,
  ListConfigCenterDriftStatesParams,
  ListConfigCenterRecoveryStatesParams,
  ListConfigCenterSpecsParams,
  UpsertConfigCenterSpecRequest,
  UpsertConfigCenterSpecResult,
} from "@/types/configCenter";

type DataEnvelope<T> = {
  data: T;
};

type DataWithTotalEnvelope<T> = {
  data: T;
  total: number;
};

export async function listConfigCenterSpecs(
  params?: ListConfigCenterSpecsParams
): Promise<ConfigCenterSpecListResponse> {
  const response = await adminApi.get<DataWithTotalEnvelope<ConfigCenterSpec[]>>("/config-center/specs", {
    params,
  });
  return {
    data: response.data.data ?? [],
    total: response.data.total ?? 0,
  };
}

export async function createConfigCenterSpec(
  payload: UpsertConfigCenterSpecRequest
): Promise<UpsertConfigCenterSpecResult> {
  const response = await adminApi.post<DataEnvelope<UpsertConfigCenterSpecResult>>(
    "/config-center/specs",
    payload
  );
  return response.data.data;
}

export async function updateConfigCenterSpec(
  specId: number,
  payload: UpsertConfigCenterSpecRequest
): Promise<UpsertConfigCenterSpecResult> {
  const response = await adminApi.put<DataEnvelope<UpsertConfigCenterSpecResult>>(
    `/config-center/specs/${specId}`,
    payload
  );
  return response.data.data;
}

export async function getConfigCenterSpecHistory(
  specId: number,
  params?: { limit?: number; offset?: number }
): Promise<ConfigCenterSpecHistoryResponse> {
  const response = await adminApi.get<DataWithTotalEnvelope<ConfigCenterSpecHistoryResponse["data"]>>(
    `/config-center/specs/${specId}/history`,
    { params }
  );
  return {
    data: response.data.data ?? [],
    total: response.data.total ?? 0,
  };
}

export async function importConfigCenterSpecsFromApplied(
  payload: ImportConfigCenterSpecRequest
): Promise<ImportConfigCenterSpecResult> {
  const response = await adminApi.post<DataEnvelope<ImportConfigCenterSpecResult>>(
    "/config-center/specs/import-from-applied",
    payload
  );
  return response.data.data;
}

export async function listConfigCenterArtifacts(
  params: ListConfigCenterArtifactsParams
): Promise<ConfigCenterArtifactListResponse> {
  const response = await adminApi.get<ConfigCenterArtifactListResponse>("/config-center/artifacts", {
    params,
  });
  return {
    desired_revision: response.data.desired_revision,
    total: response.data.total,
    data: response.data.data ?? [],
  };
}

export async function getConfigCenterTextDiff(
  params: GetConfigCenterTextDiffParams
): Promise<ConfigCenterTextDiff> {
  const response = await adminApi.get<DataEnvelope<ConfigCenterTextDiff>>("/config-center/diff/text", {
    params,
  });
  return response.data.data;
}

export async function getConfigCenterSemanticDiff(
  params: GetConfigCenterSemanticDiffParams
): Promise<ConfigCenterSemanticDiff> {
  const response = await adminApi.get<DataEnvelope<ConfigCenterSemanticDiff>>(
    "/config-center/diff/semantic",
    {
      params,
    }
  );
  return response.data.data;
}

export async function createConfigCenterApplyRun(
  payload: CreateConfigCenterApplyRunRequest
): Promise<ConfigCenterApplyRun> {
  const response = await adminApi.post<DataEnvelope<ConfigCenterApplyRun>>(
    "/config-center/apply-runs",
    payload
  );
  return response.data.data;
}

export async function listConfigCenterApplyRuns(
  params?: ListConfigCenterApplyRunsParams
): Promise<ConfigCenterApplyRunListResponse> {
  const response = await adminApi.get<DataWithTotalEnvelope<ConfigCenterApplyRun[]>>(
    "/config-center/apply-runs",
    {
      params,
    }
  );
  return {
    data: response.data.data ?? [],
    total: response.data.total ?? 0,
  };
}

export async function getConfigCenterApplyRunDetail(
  runId: string,
  params?: GetConfigCenterApplyRunDetailParams
): Promise<ConfigCenterApplyRunDetail> {
  const response = await adminApi.get<DataEnvelope<ConfigCenterApplyRunDetail>>(
    `/config-center/apply-runs/${encodeURIComponent(runId)}`,
    {
      params,
    }
  );
  return response.data.data;
}

export async function listConfigCenterAppliedSnapshot(
  params: ListConfigCenterAppliedSnapshotParams
): Promise<ConfigCenterAppliedSnapshot> {
  const response = await adminApi.get<DataEnvelope<ConfigCenterAppliedSnapshot>>(
    "/config-center/snapshot",
    {
      params,
    }
  );
  return response.data.data;
}

export async function listConfigCenterDriftStates(
  params: ListConfigCenterDriftStatesParams
): Promise<ConfigCenterDriftStateListResponse> {
  const response = await adminApi.get<DataWithTotalEnvelope<ConfigCenterDriftStateListResponse["data"]>>(
    "/config-center/drift",
    {
      params,
    }
  );
  return {
    data: response.data.data ?? [],
    total: response.data.total ?? 0,
  };
}

export async function listConfigCenterRecoveryStates(
  params: ListConfigCenterRecoveryStatesParams
): Promise<ConfigCenterDriftStateListResponse> {
  const response = await adminApi.get<DataWithTotalEnvelope<ConfigCenterDriftStateListResponse["data"]>>(
    "/config-center/recover",
    {
      params,
    }
  );
  return {
    data: response.data.data ?? [],
    total: response.data.total ?? 0,
  };
}
