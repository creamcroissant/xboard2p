import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import {
  History,
  Plus,
  RefreshCw,
  Repeat,
  Trash2,
} from "lucide-react";
import {
  listAgentCores,
  listAgentCoreInstances,
  createAgentCoreInstance,
  deleteAgentCoreInstance,
  switchAgentCore,
  listAgentCoreSwitchLogs,
} from "@/api/admin";
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
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  EmptyState,
  Input,
  Loading,
  Pagination,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui";
import type {
  AgentCoreInstance,
  AgentCoreSwitchLog,
  CreateAgentCoreInstanceRequest,
  SwitchAgentCoreRequest,
} from "@/types";

interface AgentCorePanelProps {
  agentHostId: number;
  agentName?: string;
}

type CoreInstanceForm = {
  core_type: string;
  instance_id: string;
  config_template_id: string;
};

type CoreSwitchForm = {
  from_instance_id: string;
  to_core_type: string;
  config_template_id: string;
};

const DEFAULT_FORM: CoreInstanceForm = {
  core_type: "",
  instance_id: "",
  config_template_id: "",
};

const DEFAULT_SWITCH_FORM: CoreSwitchForm = {
  from_instance_id: "",
  to_core_type: "",
  config_template_id: "",
};

const LOGS_PAGE_SIZE = 10;

function normalizeTemplateId(value: string): number | undefined {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

function formatPorts(ports: number[]): string {
  if (!ports || ports.length === 0) return "-";
  return ports.join(", ");
}

function getStatusVariant(status: string): "success" | "warning" | "danger" | "secondary" {
  switch (status) {
    case "running":
      return "success";
    case "pending":
    case "in_progress":
      return "warning";
    case "failed":
      return "danger";
    case "stopped":
      return "secondary";
    default:
      return "secondary";
  }
}

export default function AgentCorePanel({ agentHostId, agentName }: AgentCorePanelProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isSwitchOpen, setIsSwitchOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [isLogsOpen, setIsLogsOpen] = useState(false);
  const [createForm, setCreateForm] = useState<CoreInstanceForm>(DEFAULT_FORM);
  const [switchForm, setSwitchForm] = useState<CoreSwitchForm>(DEFAULT_SWITCH_FORM);
  const [deleteTarget, setDeleteTarget] = useState<AgentCoreInstance | null>(null);
  const [logsStatus, setLogsStatus] = useState<string>("");
  const [logsPage, setLogsPage] = useState(1);
  const [logsDateRange, setLogsDateRange] = useState<{ start: string; end: string }>({
    start: "",
    end: "",
  });

  const handleCreateDialogChange = (open: boolean) => {
    setIsCreateOpen(open);
    if (!open) {
      setCreateForm(DEFAULT_FORM);
    }
  };

  const handleSwitchDialogChange = (open: boolean) => {
    setIsSwitchOpen(open);
    if (!open) {
      setSwitchForm(DEFAULT_SWITCH_FORM);
    }
  };
  const coresQuery = useQuery({
    queryKey: [...QUERY_KEYS.ADMIN_AGENT_CORES, agentHostId],
    queryFn: () => listAgentCores(agentHostId),
  });

  const instancesQuery = useQuery({
    queryKey: [...QUERY_KEYS.ADMIN_AGENT_CORE_INSTANCES, agentHostId],
    queryFn: () => listAgentCoreInstances(agentHostId),
  });

  const logsQuery = useQuery({
    queryKey: [
      ...QUERY_KEYS.ADMIN_AGENT_CORE_SWITCH_LOGS,
      agentHostId,
      logsStatus,
      logsPage,
      logsDateRange.start,
      logsDateRange.end,
    ],
    queryFn: () => {
      const startAt = logsDateRange.start
        ? Math.floor(new Date(`${logsDateRange.start}T00:00:00`).getTime() / 1000)
        : undefined;
      const endAt = logsDateRange.end
        ? Math.floor(new Date(`${logsDateRange.end}T23:59:59`).getTime() / 1000)
        : undefined;
      return listAgentCoreSwitchLogs({
        agent_host_id: agentHostId,
        status: logsStatus || undefined,
        start_at: startAt,
        end_at: endAt,
        limit: LOGS_PAGE_SIZE,
        offset: (logsPage - 1) * LOGS_PAGE_SIZE,
      });
    },
    enabled: isLogsOpen,
  });

  const createMutation = useMutation({
    mutationFn: (payload: CreateAgentCoreInstanceRequest) =>
      createAgentCoreInstance(agentHostId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENT_CORE_INSTANCES });
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENT_CORE_SWITCH_LOGS });
      handleCreateDialogChange(false);
      toast.success(t("admin.cores.createSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.cores.createError"), { description: err.message });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (instanceId: string) => deleteAgentCoreInstance(agentHostId, instanceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENT_CORE_INSTANCES });
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENT_CORE_SWITCH_LOGS });
      setIsDeleteOpen(false);
      setDeleteTarget(null);
      toast.success(t("admin.cores.deleteSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.cores.deleteError"), { description: err.message });
    },
  });

  const switchMutation = useMutation({
    mutationFn: (payload: SwitchAgentCoreRequest) => switchAgentCore(agentHostId, payload),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENT_CORE_INSTANCES });
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENT_CORE_SWITCH_LOGS });
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENT_CORES });
      handleSwitchDialogChange(false);
      if (result.success) {
        toast.success(t("admin.cores.switchSuccess"));
      } else {
        toast.error(t("admin.cores.switchError"), { description: result.error || result.message });
      }
    },
    onError: (err: Error) => {
      toast.error(t("admin.cores.switchError"), { description: err.message });
    },
  });

  const handleCreate = () => {
    if (!createForm.core_type || !createForm.instance_id) {
      toast.warning(t("admin.cores.validationError"), {
        description: t("admin.cores.fieldsRequired"),
      });
      return;
    }
    const instanceId = createForm.instance_id.trim();
    if (!instanceId) {
      toast.warning(t("admin.cores.validationError"), {
        description: t("admin.cores.fieldsRequired"),
      });
      return;
    }
    createMutation.mutate({
      core_type: createForm.core_type,
      instance_id: instanceId,
      config_template_id: normalizeTemplateId(createForm.config_template_id),
    });
  };

  const handleSwitch = () => {
    if (!switchForm.from_instance_id || !switchForm.to_core_type) {
      toast.warning(t("admin.cores.validationError"), {
        description: t("admin.cores.switchFieldsRequired"),
      });
      return;
    }
    const fromInstance = switchForm.from_instance_id.trim();
    if (!fromInstance) {
      toast.warning(t("admin.cores.validationError"), {
        description: t("admin.cores.switchFieldsRequired"),
      });
      return;
    }
    switchMutation.mutate({
      from_instance_id: fromInstance,
      to_core_type: switchForm.to_core_type,
      config_template_id: normalizeTemplateId(switchForm.config_template_id),
    });
  };

  const handleDeleteRequest = (instance: AgentCoreInstance) => {
    setDeleteTarget(instance);
    setIsDeleteOpen(true);
  };

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return;
    deleteMutation.mutate(deleteTarget.instance_id);
  };

  const handleLogsDialogChange = (open: boolean) => {
    setIsLogsOpen(open);
    if (!open) {
      setLogsPage(1);
      setLogsStatus("");
      setLogsDateRange({ start: "", end: "" });
    }
  };

  const handleLogsDateChange = (next: { start?: string; end?: string }) => {
    setLogsDateRange((prev) => ({ ...prev, ...next }));
    setLogsPage(1);
  };

  const cores = coresQuery.data ?? [];
  const instances = instancesQuery.data ?? [];
  const logs = logsQuery.data?.logs ?? [];
  const logsTotal = logsQuery.data?.total ?? 0;
  const logsTotalPages = Math.ceil(logsTotal / LOGS_PAGE_SIZE);

  const installedCores = useMemo(
    () => cores.filter((core) => core.installed),
    [cores]
  );

  const coreOptions = useMemo(
    () => installedCores.map((core) => ({ value: core.type, label: core.type })),
    [installedCores]
  );

  const allCoreOptions = useMemo(
    () => cores.map((core) => ({ value: core.type, label: core.type })),
    [cores]
  );

  const instanceOptions = useMemo(
    () =>
      instances.map((instance) => ({
        value: instance.instance_id,
        label: `${instance.core_type} · ${instance.instance_id}`,
      })),
    [instances]
  );

  const availableCreateOptions = coreOptions.length > 0 ? coreOptions : allCoreOptions;

  const isLoading = coresQuery.isLoading || instancesQuery.isLoading;
  const hasError = coresQuery.error || instancesQuery.error;

  if (isLoading) {
    return <Loading />;
  }

  if (hasError) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-10">
        <p className="text-sm text-destructive">{t("admin.cores.loadError")}</p>
        <Button
          variant="outline"
          onClick={() => {
            coresQuery.refetch();
            instancesQuery.refetch();
          }}
        >
          {t("common.retry")}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid="admin-agent-core-panel">
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.cores.title")}</CardTitle>
          <CardDescription>
            {agentName
              ? t("admin.cores.subtitleWithAgent", { name: agentName })
              : t("admin.cores.subtitle")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={() => coresQuery.refetch()}>
              <RefreshCw className="mr-2 h-4 w-4" />
              {t("common.refresh")}
            </Button>
            <Button onClick={() => setIsCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t("admin.cores.create")}
            </Button>
            <Button variant="secondary" onClick={() => setIsSwitchOpen(true)}>
              <Repeat className="mr-2 h-4 w-4" />
              {t("admin.cores.switch")}
            </Button>
            <Button variant="ghost" onClick={() => setIsLogsOpen(true)}>
              <History className="mr-2 h-4 w-4" />
              {t("admin.cores.logsTitle")}
            </Button>
          </div>

          {cores.length === 0 ? (
            <EmptyState
              icon={<History className="h-full w-full" />}
              title={t("admin.cores.empty")}
              description={t("admin.cores.emptyDescription")}
              size="sm"
            />
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {cores.map((core) => (
                <Card key={core.type} className="border border-border shadow-none">
                  <CardContent className="space-y-2 p-4">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold">{core.type}</span>
                      <Badge variant={core.installed ? "success" : "secondary"}>
                        {core.installed ? t("admin.cores.installed") : t("admin.cores.uninstalled")}
                      </Badge>
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {t("admin.cores.version")}: {core.version || "-"}
                    </div>
                    {core.capabilities.length > 0 && (
                      <div className="flex flex-wrap gap-2">
                        {core.capabilities.map((capability) => (
                          <Badge key={capability} variant="secondary">
                            {capability}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.cores.instancesTitle")}</CardTitle>
          <CardDescription>{t("admin.cores.instancesSubtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          {instances.length === 0 ? (
            <EmptyState
              icon={<History className="h-full w-full" />}
              title={t("admin.cores.instancesEmpty")}
              description={t("admin.cores.instancesEmptyDescription")}
              size="sm"
            />
          ) : (
            <Table aria-label={t("admin.cores.instancesTitle")}>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("admin.cores.fields.coreType")}</TableHead>
                  <TableHead>{t("admin.cores.fields.instanceId")}</TableHead>
                  <TableHead>{t("admin.cores.fields.status")}</TableHead>
                  <TableHead>{t("admin.cores.fields.template")}</TableHead>
                  <TableHead>{t("admin.cores.fields.ports")}</TableHead>
                  <TableHead>{t("admin.cores.fields.heartbeat")}</TableHead>
                  <TableHead>{t("admin.cores.fields.error")}</TableHead>
                  <TableHead>{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {instances.map((instance) => (
                  <TableRow key={instance.instance_id}>
                    <TableCell className="font-medium">{instance.core_type}</TableCell>
                    <TableCell className="font-mono text-xs">{instance.instance_id}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusVariant(instance.status)}>{instance.status}</Badge>
                    </TableCell>
                    <TableCell>{instance.config_template_id ?? "-"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatPorts(instance.listen_ports)}
                    </TableCell>
                    <TableCell>{formatDateTime(instance.last_heartbeat_at ?? 0)}</TableCell>
                    <TableCell>
                      <div className="max-w-[200px] truncate text-xs text-muted-foreground">
                        {instance.error_message || "-"}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setSwitchForm((prev) => ({
                              ...prev,
                              from_instance_id: instance.instance_id,
                            }));
                            setIsSwitchOpen(true);
                          }}
                        >
                          {t("admin.cores.switch")}
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => handleDeleteRequest(instance)}
                        >
                          <Trash2 className="mr-1 h-3 w-3" />
                          {t("common.delete")}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={isCreateOpen} onOpenChange={handleCreateDialogChange}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{t("admin.cores.createTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.cores.fields.coreType")}</label>
              <Select
                value={createForm.core_type}
                onValueChange={(value) => setCreateForm((prev) => ({ ...prev, core_type: value }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.cores.placeholders.coreType")} />
                </SelectTrigger>
                <SelectContent>
                  {availableCreateOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.cores.fields.instanceId")}</label>
              <Input
                value={createForm.instance_id}
                onChange={(event) =>
                  setCreateForm((prev) => ({ ...prev, instance_id: event.target.value }))
                }
                placeholder={t("admin.cores.placeholders.instanceId")}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.cores.fields.template")}</label>
              <Input
                value={createForm.config_template_id}
                onChange={(event) =>
                  setCreateForm((prev) => ({ ...prev, config_template_id: event.target.value }))
                }
                placeholder={t("admin.cores.placeholders.templateId")}
              />
              <p className="text-xs text-muted-foreground">
                {t("admin.cores.templateHint")}
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => handleCreateDialogChange(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleCreate} disabled={createMutation.isPending}>
              {createMutation.isPending ? t("common.loading") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isSwitchOpen} onOpenChange={handleSwitchDialogChange}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{t("admin.cores.switchTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.cores.fields.fromInstance")}</label>
              <Select
                value={switchForm.from_instance_id}
                onValueChange={(value) => setSwitchForm((prev) => ({ ...prev, from_instance_id: value }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.cores.placeholders.fromInstance")} />
                </SelectTrigger>
                <SelectContent>
                  {instanceOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.cores.fields.coreType")}</label>
              <Select
                value={switchForm.to_core_type}
                onValueChange={(value) => setSwitchForm((prev) => ({ ...prev, to_core_type: value }))}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t("admin.cores.placeholders.coreType")} />
                </SelectTrigger>
                <SelectContent>
                  {coreOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.cores.fields.template")}</label>
              <Input
                value={switchForm.config_template_id}
                onChange={(event) =>
                  setSwitchForm((prev) => ({ ...prev, config_template_id: event.target.value }))
                }
                placeholder={t("admin.cores.placeholders.templateId")}
              />
              <p className="text-xs text-muted-foreground">
                {t("admin.cores.templateHint")}
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => handleSwitchDialogChange(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleSwitch} disabled={switchMutation.isPending}>
              {switchMutation.isPending ? t("common.loading") : t("admin.cores.switch")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isLogsOpen} onOpenChange={handleLogsDialogChange}>
        <DialogContent className="sm:max-w-5xl">
          <DialogHeader>
            <DialogTitle>{t("admin.cores.logsTitle")}</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-4 py-2">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.cores.logsStart")}</label>
                <Input
                  type="date"
                  value={logsDateRange.start}
                  onChange={(event) => handleLogsDateChange({ start: event.target.value })}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.cores.logsEnd")}</label>
                <Input
                  type="date"
                  value={logsDateRange.end}
                  onChange={(event) => handleLogsDateChange({ end: event.target.value })}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.cores.logsStatus")}</label>
                <Select value={logsStatus} onValueChange={setLogsStatus}>
                  <SelectTrigger>
                    <SelectValue placeholder={t("admin.cores.logsStatusPlaceholder")} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="">{t("common.all")}</SelectItem>
                    <SelectItem value="pending">{t("admin.cores.status.pending")}</SelectItem>
                    <SelectItem value="in_progress">{t("admin.cores.status.in_progress")}</SelectItem>
                    <SelectItem value="completed">{t("admin.cores.status.completed")}</SelectItem>
                    <SelectItem value="failed">{t("admin.cores.status.failed")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button variant="outline" onClick={() => handleLogsDateChange({ start: "", end: "" })}>
                {t("admin.cores.logsClear")}
              </Button>
            </div>

            {logsQuery.isLoading ? (
              <Loading />
            ) : logsQuery.error ? (
              <div className="flex flex-col items-center justify-center gap-3 py-6">
                <p className="text-sm text-destructive">{t("admin.cores.logsLoadError")}</p>
                <Button variant="outline" onClick={() => logsQuery.refetch()}>
                  {t("common.retry")}
                </Button>
              </div>
            ) : logs.length === 0 ? (
              <EmptyState
                icon={<History className="h-full w-full" />}
                title={t("admin.cores.logsEmpty")}
                description={t("admin.cores.logsEmptyDescription")}
                size="sm"
              />
            ) : (
              <Table aria-label={t("admin.cores.logsTitle")}>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("admin.cores.logsTime")}</TableHead>
                    <TableHead>{t("admin.cores.logsFrom")}</TableHead>
                    <TableHead>{t("admin.cores.logsTo")}</TableHead>
                    <TableHead>{t("admin.cores.logsStatus")}</TableHead>
                    <TableHead>{t("admin.cores.logsOperator")}</TableHead>
                    <TableHead>{t("admin.cores.logsDetail")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log: AgentCoreSwitchLog) => (
                    <TableRow key={log.id}>
                      <TableCell className="whitespace-nowrap">{formatDateTime(log.created_at)}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {log.from_core_type || "-"}
                        <div className="font-mono">{log.from_instance_id || "-"}</div>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {log.to_core_type}
                        <div className="font-mono">{log.to_instance_id}</div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={getStatusVariant(log.status)}>{log.status}</Badge>
                      </TableCell>
                      <TableCell>{log.operator_id ?? "-"}</TableCell>
                      <TableCell>
                        <div className="max-w-xl truncate text-xs text-muted-foreground">
                          {log.detail || "-"}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}

            <Pagination page={logsPage} totalPages={logsTotalPages} onPageChange={setLogsPage} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => handleLogsDialogChange(false)}>
              {t("common.close")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("admin.cores.deleteTitle")}</DialogTitle>
          </DialogHeader>
          <div className="py-2 text-sm text-muted-foreground">
            {t("admin.cores.deleteConfirm", {
              instanceId: deleteTarget?.instance_id,
              coreType: deleteTarget?.core_type,
            })}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDeleteConfirm} disabled={deleteMutation.isPending}>
              {deleteMutation.isPending ? t("common.loading") : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
