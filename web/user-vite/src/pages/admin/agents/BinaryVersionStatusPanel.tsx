import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { listAgentBinaryVersions, refreshAgentBinaryVersion } from "@/api/admin";
import { QUERY_KEYS } from "@/lib/constants";
import { formatDateTime } from "@/lib/format";
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Loading,
} from "@/components/ui";
import type { BinaryVersionComponent, BinaryVersionState, BinaryVersionStatus } from "@/types";

interface BinaryVersionStatusPanelProps {
  agentHostId: number;
}

const COMPONENTS: BinaryVersionComponent[] = ["agent", "sing-box", "xray"];

function getVersionStatusVariant(status: BinaryVersionStatus): "success" | "warning" | "danger" | "secondary" {
  switch (status) {
    case "up_to_date":
    case "installed":
      return "success";
    case "outdated":
      return "warning";
    case "missing":
      return "danger";
    default:
      return "secondary";
  }
}

function normalizeVersion(value?: string): string {
  const trimmed = value?.trim();
  return trimmed || "-";
}

function buildRows(states: BinaryVersionState[]): BinaryVersionState[] {
  const byComponent = new Map(states.map((state) => [state.component, state]));
  return COMPONENTS.map((component) =>
    byComponent.get(component) ?? {
      agent_host_id: 0,
      component,
      local_version: "",
      status: "unknown",
    }
  );
}

export function BinaryVersionStatusPanel({ agentHostId }: BinaryVersionStatusPanelProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const versionsQuery = useQuery({
    queryKey: [...QUERY_KEYS.ADMIN_AGENT_BINARY_VERSIONS, agentHostId],
    queryFn: () => listAgentBinaryVersions(agentHostId),
  });

  const refreshMutation = useMutation({
    mutationFn: (component: BinaryVersionComponent) => refreshAgentBinaryVersion(agentHostId, component),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENT_BINARY_VERSIONS });
      toast.success(t("admin.cores.versionRefreshSuccess"));
    },
    onError: (error: Error) => {
      toast.error(t("admin.cores.versionRefreshError"), { description: error.message });
    },
  });

  const rows = buildRows(versionsQuery.data ?? []);

  return (
    <Card className="border border-border shadow-none">
      <CardHeader className="pb-3">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle className="text-base">{t("admin.cores.versionStatusTitle")}</CardTitle>
            <CardDescription>{t("admin.cores.versionStatusDescription")}</CardDescription>
          </div>
          <Button
            size="sm"
            variant="outline"
            onClick={() => versionsQuery.refetch()}
            disabled={versionsQuery.isFetching}
          >
            <RefreshCw className="mr-2 h-3.5 w-3.5" />
            {t("common.refresh")}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {versionsQuery.isLoading ? (
          <Loading />
        ) : versionsQuery.error ? (
          <div className="flex flex-col items-center justify-center gap-3 py-6">
            <p className="text-sm text-destructive">{t("admin.cores.versionLoadError")}</p>
            <Button variant="outline" onClick={() => versionsQuery.refetch()}>
              {t("common.retry")}
            </Button>
          </div>
        ) : rows.length === 0 ? (
          <EmptyState
            icon={<RefreshCw className="h-full w-full" />}
            title={t("admin.cores.versionEmpty")}
            description={t("admin.cores.versionEmptyDescription")}
            size="sm"
          />
        ) : (
          <div className="grid gap-3 lg:grid-cols-3">
            {rows.map((state) => {
              const isRefreshing = refreshMutation.isPending && refreshMutation.variables === state.component;
              return (
                <div key={state.component} className="rounded-md border border-border p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="font-semibold">{t(`admin.cores.binaryComponent.${state.component}`)}</div>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {t("admin.cores.versionCheckedAt")}: {formatDateTime(state.last_checked_at ?? 0)}
                      </div>
                    </div>
                    <Badge variant={getVersionStatusVariant(state.status)}>
                      {t(`admin.cores.versionState.${state.status}`)}
                    </Badge>
                  </div>
                  <div className="mt-3 grid gap-2 text-sm">
                    <div className="flex justify-between gap-3">
                      <span className="text-muted-foreground">{t("admin.cores.versionLocal")}</span>
                      <span className="font-mono text-xs">{normalizeVersion(state.local_version)}</span>
                    </div>
                    <div className="flex justify-between gap-3">
                      <span className="text-muted-foreground">{t("admin.cores.versionRemote")}</span>
                      <span className="font-mono text-xs">{normalizeVersion(state.remote_version)}</span>
                    </div>
                  </div>
                  {state.last_check_error && (
                    <div className="mt-3 rounded-md border border-warning/30 bg-warning/10 p-2 text-xs text-warning-foreground dark:text-warning">
                      {state.last_check_error}
                    </div>
                  )}
                  <Button
                    size="sm"
                    variant="outline"
                    className="mt-3"
                    onClick={() => refreshMutation.mutate(state.component)}
                    disabled={isRefreshing}
                  >
                    <RefreshCw className="mr-2 h-3.5 w-3.5" />
                    {isRefreshing ? t("common.loading") : t("admin.cores.versionRefresh")}
                  </Button>
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
