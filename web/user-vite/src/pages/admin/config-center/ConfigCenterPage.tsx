import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  CheckCircle2,
  Diff,
  GitCompare,
  GitCommitHorizontal,
  History,
  RefreshCw,
  ShieldAlert,
  Upload,
} from "lucide-react";
import { QUERY_KEYS } from "@/lib/constants";
import {
  createConfigCenterApplyRun,
  createConfigCenterSpec,
  getConfigCenterSemanticDiff,
  getConfigCenterSpecHistory,
  getConfigCenterTextDiff,
  importConfigCenterSpecsFromApplied,
  listConfigCenterAppliedSnapshot,
  listConfigCenterDriftStates,
  listConfigCenterRecoveryStates,
  listConfigCenterSpecs,
  updateConfigCenterSpec,
} from "@/api/admin";
import { getAgentHosts } from "@/api/admin/agentHost";
import { formatDateTime } from "@/lib/format";
import type {
  AgentHost,
  ConfigCenterAppliedSnapshot,
  ConfigCenterCoreType,
  ConfigCenterSpec,
  ConfigCenterSpecRevision,
  CreateConfigCenterApplyRunRequest,
  ImportConfigCenterSpecRequest,
  UpsertConfigCenterSpecRequest,
} from "@/types";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Textarea,
} from "@/components/ui";

type CoreTypeOption = ConfigCenterCoreType;

type SpecFormState = {
  agent_host_id: number;
  core_type: CoreTypeOption;
  tag: string;
  enabled: boolean;
  semantic_spec: string;
  core_specific: string;
  change_note: string;
};

type ImportFormState = {
  source: "legacy" | "managed" | "merged";
  filename: string;
  tag: string;
  enabled: boolean;
  overwrite_existing: boolean;
  change_note: string;
};

type ApplyFormState = {
  target_revision: string;
  previous_revision: string;
};

const CORE_OPTIONS: CoreTypeOption[] = ["sing-box", "xray"];

const defaultSpecFormState: SpecFormState = {
  agent_host_id: 0,
  core_type: "sing-box",
  tag: "",
  enabled: true,
  semantic_spec: "{}",
  core_specific: "{}",
  change_note: "",
};

const defaultImportFormState: ImportFormState = {
  source: "merged",
  filename: "",
  tag: "",
  enabled: true,
  overwrite_existing: true,
  change_note: "",
};

const defaultApplyFormState: ApplyFormState = {
  target_revision: "",
  previous_revision: "",
};

function safeParseJSON(input: string, fallback: unknown = {}): unknown {
  const text = input.trim();
  if (!text) return fallback;
  return JSON.parse(text);
}

function prettyJSON(input: unknown): string {
  try {
    return JSON.stringify(input ?? {}, null, 2);
  } catch {
    return "{}";
  }
}

function formatCoreType(coreType: ConfigCenterCoreType): CoreTypeOption {
  if (coreType === "xray") {
    return "xray";
  }
  return "sing-box";
}


function formatDriftVariant(driftType: string): "danger" | "warning" | "secondary" {
  switch (driftType) {
    case "hash_mismatch":
    case "tag_conflict":
      return "danger";
    case "missing_tag":
    case "parse_error":
      return "warning";
    default:
      return "secondary";
  }
}

export default function ConfigCenterPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [selectedHostId, setSelectedHostId] = useState<number | null>(null);
  const [selectedCoreType, setSelectedCoreType] = useState<CoreTypeOption>("sing-box");
  const [selectedSpec, setSelectedSpec] = useState<ConfigCenterSpec | null>(null);

  const [isSpecDialogOpen, setIsSpecDialogOpen] = useState(false);
  const [isHistoryDialogOpen, setIsHistoryDialogOpen] = useState(false);
  const [isImportDialogOpen, setIsImportDialogOpen] = useState(false);

  const [specForm, setSpecForm] = useState<SpecFormState>(defaultSpecFormState);
  const [importForm, setImportForm] = useState<ImportFormState>(defaultImportFormState);
  const [applyForm, setApplyForm] = useState<ApplyFormState>(defaultApplyFormState);

  const [historyTarget, setHistoryTarget] = useState<ConfigCenterSpec | null>(null);

  const [diffFilename, setDiffFilename] = useState("");
  const [diffTag, setDiffTag] = useState("");
  const [diffRevision, setDiffRevision] = useState("");

  const hostQuery = useQuery({
    queryKey: QUERY_KEYS.ADMIN_AGENTS,
    queryFn: () => getAgentHosts({ page: 1, page_size: 100 }),
  });

  const specListQuery = useQuery({
    queryKey: [
      ...QUERY_KEYS.ADMIN_CONFIG_CENTER_SPECS,
      selectedHostId,
      selectedCoreType,
    ],
    queryFn: () =>
      listConfigCenterSpecs({
        agent_host_id: selectedHostId ?? undefined,
        core_type: selectedCoreType,
        limit: 100,
        offset: 0,
      }),
    enabled: selectedHostId !== null,
  });

  const historyQuery = useQuery({
    queryKey: [
      ...QUERY_KEYS.ADMIN_CONFIG_CENTER_SPEC_HISTORY,
      historyTarget?.id,
    ],
    queryFn: () => getConfigCenterSpecHistory(historyTarget?.id ?? 0, { limit: 50, offset: 0 }),
    enabled: Boolean(historyTarget?.id) && isHistoryDialogOpen,
  });

  const snapshotQuery = useQuery({
    queryKey: [
      ...QUERY_KEYS.ADMIN_CONFIG_CENTER_SNAPSHOT,
      selectedHostId,
      selectedCoreType,
    ],
    queryFn: () =>
      listConfigCenterAppliedSnapshot({
        agent_host_id: selectedHostId ?? 0,
        core_type: selectedCoreType,
        limit: 200,
        offset: 0,
      }),
    enabled: selectedHostId !== null,
  });

  const driftQuery = useQuery({
    queryKey: [
      ...QUERY_KEYS.ADMIN_CONFIG_CENTER_DRIFT,
      selectedHostId,
      selectedCoreType,
    ],
    queryFn: () =>
      listConfigCenterDriftStates({
        agent_host_id: selectedHostId ?? 0,
        core_type: selectedCoreType,
        limit: 200,
        offset: 0,
      }),
    enabled: selectedHostId !== null,
  });

  const recoveryQuery = useQuery({
    queryKey: [
      ...QUERY_KEYS.ADMIN_CONFIG_CENTER_RECOVER,
      selectedHostId,
      selectedCoreType,
    ],
    queryFn: () =>
      listConfigCenterRecoveryStates({
        agent_host_id: selectedHostId ?? 0,
        core_type: selectedCoreType,
        limit: 200,
        offset: 0,
      }),
    enabled: selectedHostId !== null,
  });

  const textDiffQuery = useQuery({
    queryKey: [
      ...QUERY_KEYS.ADMIN_CONFIG_CENTER_DIFF_TEXT,
      selectedHostId,
      selectedCoreType,
      diffRevision,
      diffFilename,
      diffTag,
    ],
    queryFn: () =>
      getConfigCenterTextDiff({
        agent_host_id: selectedHostId ?? 0,
        core_type: selectedCoreType,
        desired_revision: diffRevision ? Number(diffRevision) : undefined,
        filename: diffFilename.trim() || undefined,
        tag: diffTag.trim() || undefined,
      }),
    enabled: selectedHostId !== null,
  });

  const semanticDiffQuery = useQuery({
    queryKey: [
      ...QUERY_KEYS.ADMIN_CONFIG_CENTER_DIFF_SEMANTIC,
      selectedHostId,
      selectedCoreType,
      diffRevision,
      diffTag,
    ],
    queryFn: () =>
      getConfigCenterSemanticDiff({
        agent_host_id: selectedHostId ?? 0,
        core_type: selectedCoreType,
        desired_revision: diffRevision ? Number(diffRevision) : undefined,
        tag: diffTag.trim() || undefined,
      }),
    enabled: selectedHostId !== null,
  });

  const selectedHost = useMemo<AgentHost | null>(() => {
    const hosts = hostQuery.data?.data ?? [];
    return hosts.find((host) => host.id === selectedHostId) ?? null;
  }, [hostQuery.data, selectedHostId]);

  const specs = specListQuery.data?.data ?? [];
  const snapshot = snapshotQuery.data as ConfigCenterAppliedSnapshot | undefined;
  const driftStates = driftQuery.data?.data ?? [];
  const recoveryStates = recoveryQuery.data?.data ?? [];
  const historyItems = historyQuery.data?.data ?? [];

  const latestDesiredRevision = useMemo(() => {
    if (specs.length === 0) return 0;
    return specs.reduce((max, item) => Math.max(max, item.desired_revision), 0);
  }, [specs]);

  const createSpecMutation = useMutation({
    mutationFn: createConfigCenterSpec,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_CONFIG_CENTER_SPECS });
      setIsSpecDialogOpen(false);
      setSelectedSpec(null);
      setSpecForm((prev) => ({
        ...defaultSpecFormState,
        agent_host_id: prev.agent_host_id,
        core_type: prev.core_type,
      }));
      toast.success(t("admin.configCenter.messages.specSaved"));
    },
    onError: (error: Error) => {
      toast.error(t("admin.configCenter.messages.specSaveFailed"), {
        description: error.message,
      });
    },
  });

  const updateSpecMutation = useMutation({
    mutationFn: ({ specId, payload }: { specId: number; payload: UpsertConfigCenterSpecRequest }) =>
      updateConfigCenterSpec(specId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_CONFIG_CENTER_SPECS });
      setIsSpecDialogOpen(false);
      setSelectedSpec(null);
      toast.success(t("admin.configCenter.messages.specUpdated"));
    },
    onError: (error: Error) => {
      toast.error(t("admin.configCenter.messages.specSaveFailed"), {
        description: error.message,
      });
    },
  });

  const importMutation = useMutation({
    mutationFn: importConfigCenterSpecsFromApplied,
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_CONFIG_CENTER_SPECS });
      setIsImportDialogOpen(false);
      toast.success(t("admin.configCenter.messages.importSuccess", { count: result.created_count }));
    },
    onError: (error: Error) => {
      toast.error(t("admin.configCenter.messages.importFailed"), {
        description: error.message,
      });
    },
  });

  const applyMutation = useMutation({
    mutationFn: createConfigCenterApplyRun,
    onSuccess: (run) => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_CONFIG_CENTER_APPLY_RUNS });
      toast.success(t("admin.configCenter.messages.applyStarted"), {
        description: `${run.run_id}`,
      });
    },
    onError: (error: Error) => {
      toast.error(t("admin.configCenter.messages.applyFailed"), {
        description: error.message,
      });
    },
  });

  const openCreateDialog = () => {
    if (!selectedHostId) {
      toast.warning(t("admin.configCenter.messages.selectHostFirst"));
      return;
    }
    setSelectedSpec(null);
    setSpecForm({
      ...defaultSpecFormState,
      agent_host_id: selectedHostId,
      core_type: selectedCoreType,
    });
    setIsSpecDialogOpen(true);
  };

  const openEditDialog = (spec: ConfigCenterSpec) => {
    setSelectedSpec(spec);
    setSpecForm({
      agent_host_id: spec.agent_host_id,
      core_type: formatCoreType(spec.core_type),
      tag: spec.tag,
      enabled: spec.enabled,
      semantic_spec: prettyJSON(spec.semantic_spec),
      core_specific: prettyJSON(spec.core_specific),
      change_note: "",
    });
    setIsSpecDialogOpen(true);
  };

  const openHistoryDialog = (spec: ConfigCenterSpec) => {
    setHistoryTarget(spec);
    setIsHistoryDialogOpen(true);
  };

  const handleSaveSpec = () => {
    try {
      if (!specForm.agent_host_id || !specForm.tag.trim()) {
        toast.warning(t("admin.configCenter.messages.requiredFields"));
        return;
      }

      const payload: UpsertConfigCenterSpecRequest = {
        agent_host_id: specForm.agent_host_id,
        core_type: specForm.core_type,
        tag: specForm.tag.trim(),
        enabled: specForm.enabled,
        semantic_spec: safeParseJSON(specForm.semantic_spec, {}),
        core_specific: safeParseJSON(specForm.core_specific, {}),
        change_note: specForm.change_note.trim() || undefined,
      };

      if (selectedSpec) {
        updateSpecMutation.mutate({ specId: selectedSpec.id, payload });
      } else {
        createSpecMutation.mutate(payload);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : t("error.bad_request");
      toast.error(t("admin.configCenter.messages.invalidJson"), { description: message });
    }
  };

  const handleImport = () => {
    if (!selectedHostId) {
      toast.warning(t("admin.configCenter.messages.selectHostFirst"));
      return;
    }

    const payload: ImportConfigCenterSpecRequest = {
      agent_host_id: selectedHostId,
      core_type: selectedCoreType,
      source: importForm.source,
      filename: importForm.filename.trim() || undefined,
      tag: importForm.tag.trim() || undefined,
      enabled: importForm.enabled,
      change_note: importForm.change_note.trim() || undefined,
      overwrite_existing: importForm.overwrite_existing,
    };

    importMutation.mutate(payload);
  };

  const handleApply = () => {
    if (!selectedHostId) {
      toast.warning(t("admin.configCenter.messages.selectHostFirst"));
      return;
    }
    const targetRevision = Number(applyForm.target_revision);
    if (!Number.isFinite(targetRevision) || targetRevision <= 0) {
      toast.warning(t("admin.configCenter.messages.invalidTargetRevision"));
      return;
    }

    const payload: CreateConfigCenterApplyRunRequest = {
      agent_host_id: selectedHostId,
      core_type: selectedCoreType,
      target_revision: targetRevision,
      previous_revision:
        applyForm.previous_revision.trim() && Number(applyForm.previous_revision) > 0
          ? Number(applyForm.previous_revision)
          : undefined,
    };

    applyMutation.mutate(payload);
  };

  const handleHostChange = (value: string) => {
    const parsed = Number(value);
    if (Number.isFinite(parsed) && parsed > 0) {
      setSelectedHostId(parsed);
      setSelectedSpec(null);
      setApplyForm(defaultApplyFormState);
    }
  };

  const handleCoreTypeChange = (value: string) => {
    const next = value as CoreTypeOption;
    setSelectedCoreType(next);
    setSelectedSpec(null);
    setApplyForm(defaultApplyFormState);
  };

  if (hostQuery.isLoading) {
    return <Loading />;
  }

  if (hostQuery.error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <p className="text-sm text-destructive">{t("admin.configCenter.messages.hostLoadFailed")}</p>
        <Button variant="outline" onClick={() => hostQuery.refetch()}>
          {t("common.retry")}
        </Button>
      </div>
    );
  }

  const hosts = hostQuery.data?.data ?? [];

  return (
    <div className="space-y-6" data-testid="admin-config-center-page">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{t("admin.configCenter.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("admin.configCenter.subtitle")}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => {
            specListQuery.refetch();
            snapshotQuery.refetch();
            driftQuery.refetch();
            recoveryQuery.refetch();
            textDiffQuery.refetch();
            semanticDiffQuery.refetch();
          }}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {t("common.refresh")}
          </Button>
          <Button variant="secondary" onClick={() => setIsImportDialogOpen(true)} disabled={!selectedHostId}>
            <Upload className="mr-2 h-4 w-4" />
            {t("admin.configCenter.actions.import")}
          </Button>
          <Button onClick={openCreateDialog} disabled={!selectedHostId}>
            <GitCommitHorizontal className="mr-2 h-4 w-4" />
            {t("admin.configCenter.actions.createSpec")}
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("admin.configCenter.filters.title")}</CardTitle>
          <CardDescription>{t("admin.configCenter.filters.description")}</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-3">
          <div className="space-y-2">
            <label className="text-sm font-medium">{t("admin.configCenter.fields.agentHost")}</label>
            <Select value={selectedHostId ? String(selectedHostId) : undefined} onValueChange={handleHostChange}>
              <SelectTrigger>
                <SelectValue placeholder={t("admin.configCenter.placeholders.selectHost")} />
              </SelectTrigger>
              <SelectContent>
                {hosts.map((host) => (
                  <SelectItem key={String(host.id)} value={String(host.id)}>
                    {host.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">{t("admin.configCenter.fields.coreType")}</label>
            <Select value={selectedCoreType} onValueChange={handleCoreTypeChange}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CORE_OPTIONS.map((item) => (
                  <SelectItem key={item} value={item}>
                    {item}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">{t("admin.configCenter.fields.selectedHost")}</label>
            <div className="rounded-md border border-border px-3 py-2 text-sm text-muted-foreground">
              {selectedHost ? `${selectedHost.name} (${selectedHost.host})` : t("admin.configCenter.placeholders.noHost")}
            </div>
          </div>
        </CardContent>
      </Card>

      {!selectedHostId ? (
        <EmptyState
          icon={<GitCommitHorizontal className="h-full w-full" />}
          title={t("admin.configCenter.empty.noHostTitle")}
          description={t("admin.configCenter.empty.noHostDescription")}
          size="md"
        />
      ) : (
        <Tabs defaultValue="specs">
          <TabsList className="flex w-full flex-wrap justify-start">
            <TabsTrigger value="specs">{t("admin.configCenter.tabs.specs")}</TabsTrigger>
            <TabsTrigger value="diff">{t("admin.configCenter.tabs.diff")}</TabsTrigger>
            <TabsTrigger value="drift">{t("admin.configCenter.tabs.drift")}</TabsTrigger>
            <TabsTrigger value="snapshot">{t("admin.configCenter.tabs.snapshot")}</TabsTrigger>
          </TabsList>

          <TabsContent value="specs" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>{t("admin.configCenter.specs.title")}</CardTitle>
                <CardDescription>
                  {t("admin.configCenter.specs.description", { count: specs.length, revision: latestDesiredRevision })}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-3 md:grid-cols-[1fr_1fr_auto] md:items-end">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">{t("admin.configCenter.apply.targetRevision")}</label>
                    <Input
                      value={applyForm.target_revision}
                      onChange={(event) => setApplyForm((prev) => ({ ...prev, target_revision: event.target.value }))}
                      placeholder={latestDesiredRevision > 0 ? String(latestDesiredRevision) : "1"}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">{t("admin.configCenter.apply.previousRevision")}</label>
                    <Input
                      value={applyForm.previous_revision}
                      onChange={(event) =>
                        setApplyForm((prev) => ({ ...prev, previous_revision: event.target.value }))
                      }
                      placeholder={t("admin.configCenter.placeholders.optional")}
                    />
                  </div>
                  <Button onClick={handleApply} disabled={applyMutation.isPending || specs.length === 0}>
                    <CheckCircle2 className="mr-2 h-4 w-4" />
                    {applyMutation.isPending
                      ? t("common.loading")
                      : t("admin.configCenter.actions.apply")}
                  </Button>
                </div>

                {specListQuery.isLoading ? (
                  <Loading />
                ) : specListQuery.error ? (
                  <div className="flex flex-col items-center justify-center gap-3 py-10">
                    <p className="text-sm text-destructive">{t("admin.configCenter.messages.specLoadFailed")}</p>
                    <Button variant="outline" onClick={() => specListQuery.refetch()}>
                      {t("common.retry")}
                    </Button>
                  </div>
                ) : specs.length === 0 ? (
                  <EmptyState
                    icon={<GitCommitHorizontal className="h-full w-full" />}
                    title={t("admin.configCenter.empty.noSpecTitle")}
                    description={t("admin.configCenter.empty.noSpecDescription")}
                    action={<Button onClick={openCreateDialog}>{t("admin.configCenter.actions.createSpec")}</Button>}
                    size="sm"
                  />
                ) : (
                  <Table aria-label={t("admin.configCenter.specs.title") as string}>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("admin.configCenter.fields.tag")}</TableHead>
                        <TableHead>{t("admin.configCenter.fields.coreType")}</TableHead>
                        <TableHead>{t("admin.configCenter.fields.revision")}</TableHead>
                        <TableHead>{t("admin.configCenter.fields.enabled")}</TableHead>
                        <TableHead>{t("admin.configCenter.fields.updatedAt")}</TableHead>
                        <TableHead>{t("common.actions")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {specs.map((spec) => (
                        <TableRow key={spec.id}>
                          <TableCell className="font-medium">{spec.tag}</TableCell>
                          <TableCell>
                            <Badge variant="secondary">{formatCoreType(spec.core_type)}</Badge>
                          </TableCell>
                          <TableCell>{spec.desired_revision}</TableCell>
                          <TableCell>
                            <Badge variant={spec.enabled ? "success" : "secondary"}>
                              {spec.enabled
                                ? t("admin.configCenter.status.enabled")
                                : t("admin.configCenter.status.disabled")}
                            </Badge>
                          </TableCell>
                          <TableCell>{formatDateTime(spec.updated_at)}</TableCell>
                          <TableCell>
                            <div className="flex flex-wrap gap-2">
                              <Button size="sm" variant="outline" onClick={() => openEditDialog(spec)}>
                                {t("common.edit")}
                              </Button>
                              <Button size="sm" variant="ghost" onClick={() => openHistoryDialog(spec)}>
                                <History className="mr-1 h-3 w-3" />
                                {t("admin.configCenter.actions.history")}
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
          </TabsContent>

          <TabsContent value="diff" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>{t("admin.configCenter.diff.title")}</CardTitle>
                <CardDescription>{t("admin.configCenter.diff.description")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-3 md:grid-cols-3">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">{t("admin.configCenter.fields.revision")}</label>
                    <Input
                      value={diffRevision}
                      onChange={(event) => setDiffRevision(event.target.value)}
                      placeholder={t("admin.configCenter.placeholders.optional")}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">{t("admin.configCenter.fields.filename")}</label>
                    <Input
                      value={diffFilename}
                      onChange={(event) => setDiffFilename(event.target.value)}
                      placeholder={t("admin.configCenter.placeholders.optional")}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">{t("admin.configCenter.fields.tag")}</label>
                    <Input
                      value={diffTag}
                      onChange={(event) => setDiffTag(event.target.value)}
                      placeholder={t("admin.configCenter.placeholders.optional")}
                    />
                  </div>
                </div>

                <Tabs defaultValue="text">
                  <TabsList>
                    <TabsTrigger value="text">{t("admin.configCenter.diff.text")}</TabsTrigger>
                    <TabsTrigger value="semantic">{t("admin.configCenter.diff.semantic")}</TabsTrigger>
                  </TabsList>

                  <TabsContent value="text" className="space-y-3">
                    {textDiffQuery.isLoading ? (
                      <Loading />
                    ) : textDiffQuery.error ? (
                      <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                        {t("admin.configCenter.messages.textDiffFailed")}
                      </div>
                    ) : textDiffQuery.data ? (
                      <>
                        <div className="grid gap-3 md:grid-cols-2">
                          <div className="rounded-md border border-border p-3">
                            <p className="mb-2 text-xs text-muted-foreground">{t("admin.configCenter.diff.desired")}</p>
                            <pre className="max-h-60 overflow-auto whitespace-pre-wrap text-xs">
                              {textDiffQuery.data.desired_text}
                            </pre>
                          </div>
                          <div className="rounded-md border border-border p-3">
                            <p className="mb-2 text-xs text-muted-foreground">{t("admin.configCenter.diff.applied")}</p>
                            <pre className="max-h-60 overflow-auto whitespace-pre-wrap text-xs">
                              {textDiffQuery.data.applied_text}
                            </pre>
                          </div>
                        </div>
                        <div className="rounded-md border border-border p-3">
                          <p className="mb-2 text-xs text-muted-foreground">
                            {t("admin.configCenter.diff.unified")}
                          </p>
                          <pre className="max-h-80 overflow-auto whitespace-pre-wrap text-xs">
                            {textDiffQuery.data.unified_diff || "-"}
                          </pre>
                        </div>
                      </>
                    ) : null}
                  </TabsContent>

                  <TabsContent value="semantic">
                    {semanticDiffQuery.isLoading ? (
                      <Loading />
                    ) : semanticDiffQuery.error ? (
                      <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                        {t("admin.configCenter.messages.semanticDiffFailed")}
                      </div>
                    ) : semanticDiffQuery.data ? (
                      <Table aria-label={t("admin.configCenter.diff.semantic") as string}>
                        <TableHeader>
                          <TableRow>
                            <TableHead>{t("admin.configCenter.fields.tag")}</TableHead>
                            <TableHead>{t("admin.configCenter.fields.driftType")}</TableHead>
                            <TableHead>{t("admin.configCenter.fields.fieldDiffs")}</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {semanticDiffQuery.data.items.length === 0 ? (
                            <TableRow>
                              <TableCell colSpan={3} className="text-center text-muted-foreground">
                                {t("admin.configCenter.empty.noSemanticDiff")}
                              </TableCell>
                            </TableRow>
                          ) : (
                            semanticDiffQuery.data.items.map((item, index) => (
                              <TableRow key={`${item.tag}-${index}`}>
                                <TableCell>{item.tag}</TableCell>
                                <TableCell>
                                  <Badge variant={formatDriftVariant(item.drift_type)}>{item.drift_type}</Badge>
                                </TableCell>
                                <TableCell>
                                  {item.field_diffs && item.field_diffs.length > 0 ? (
                                    <div className="space-y-1 text-xs text-muted-foreground">
                                      {item.field_diffs.map((fieldDiff, fdIndex) => (
                                        <div key={`${fieldDiff.field}-${fdIndex}`}>
                                          <span className="font-medium text-foreground">{fieldDiff.field}</span>
                                          <span className="mx-1">:</span>
                                          <span>{fieldDiff.desired}</span>
                                          <span className="mx-1">→</span>
                                          <span>{fieldDiff.applied}</span>
                                        </div>
                                      ))}
                                    </div>
                                  ) : (
                                    <span className="text-xs text-muted-foreground">-</span>
                                  )}
                                </TableCell>
                              </TableRow>
                            ))
                          )}
                        </TableBody>
                      </Table>
                    ) : null}
                  </TabsContent>
                </Tabs>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="drift" className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle>{t("admin.configCenter.drift.activeTitle")}</CardTitle>
                  <CardDescription>{t("admin.configCenter.drift.activeDescription")}</CardDescription>
                </CardHeader>
                <CardContent>
                  {driftQuery.isLoading ? (
                    <Loading />
                  ) : driftQuery.error ? (
                    <p className="text-sm text-destructive">{t("admin.configCenter.messages.driftLoadFailed")}</p>
                  ) : driftStates.length === 0 ? (
                    <EmptyState
                      icon={<ShieldAlert className="h-full w-full" />}
                      title={t("admin.configCenter.empty.noDriftTitle")}
                      description={t("admin.configCenter.empty.noDriftDescription")}
                      size="sm"
                    />
                  ) : (
                    <Table aria-label={t("admin.configCenter.drift.activeTitle") as string}>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t("admin.configCenter.fields.tag")}</TableHead>
                          <TableHead>{t("admin.configCenter.fields.filename")}</TableHead>
                          <TableHead>{t("admin.configCenter.fields.driftType")}</TableHead>
                          <TableHead>{t("admin.configCenter.fields.updatedAt")}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {driftStates.map((item) => (
                          <TableRow key={item.id}>
                            <TableCell>{item.tag}</TableCell>
                            <TableCell>{item.filename}</TableCell>
                            <TableCell>
                              <Badge variant={formatDriftVariant(item.drift_type)}>{item.drift_type}</Badge>
                            </TableCell>
                            <TableCell>{formatDateTime(item.last_changed_at)}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>{t("admin.configCenter.drift.recoveredTitle")}</CardTitle>
                  <CardDescription>{t("admin.configCenter.drift.recoveredDescription")}</CardDescription>
                </CardHeader>
                <CardContent>
                  {recoveryQuery.isLoading ? (
                    <Loading />
                  ) : recoveryQuery.error ? (
                    <p className="text-sm text-destructive">{t("admin.configCenter.messages.recoveryLoadFailed")}</p>
                  ) : recoveryStates.length === 0 ? (
                    <EmptyState
                      icon={<CheckCircle2 className="h-full w-full" />}
                      title={t("admin.configCenter.empty.noRecoveryTitle")}
                      description={t("admin.configCenter.empty.noRecoveryDescription")}
                      size="sm"
                    />
                  ) : (
                    <Table aria-label={t("admin.configCenter.drift.recoveredTitle") as string}>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t("admin.configCenter.fields.tag")}</TableHead>
                          <TableHead>{t("admin.configCenter.fields.filename")}</TableHead>
                          <TableHead>{t("admin.configCenter.fields.driftType")}</TableHead>
                          <TableHead>{t("admin.configCenter.fields.updatedAt")}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {recoveryStates.map((item) => (
                          <TableRow key={item.id}>
                            <TableCell>{item.tag}</TableCell>
                            <TableCell>{item.filename}</TableCell>
                            <TableCell>
                              <Badge variant="secondary">{item.drift_type}</Badge>
                            </TableCell>
                            <TableCell>{formatDateTime(item.last_changed_at)}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value="snapshot" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>{t("admin.configCenter.snapshot.title")}</CardTitle>
                <CardDescription>{t("admin.configCenter.snapshot.description")}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                {snapshotQuery.isLoading ? (
                  <Loading />
                ) : snapshotQuery.error ? (
                  <p className="text-sm text-destructive">{t("admin.configCenter.messages.snapshotLoadFailed")}</p>
                ) : (
                  <>
                    <div>
                      <h3 className="mb-3 text-sm font-semibold text-foreground">
                        {t("admin.configCenter.snapshot.inventoryTitle")}
                      </h3>
                      {!snapshot || snapshot.inventories.length === 0 ? (
                        <EmptyState
                          icon={<Diff className="h-full w-full" />}
                          title={t("admin.configCenter.empty.noInventoryTitle")}
                          description={t("admin.configCenter.empty.noInventoryDescription")}
                          size="sm"
                        />
                      ) : (
                        <Table aria-label={t("admin.configCenter.snapshot.inventoryTitle") as string}>
                          <TableHeader>
                            <TableRow>
                              <TableHead>{t("admin.configCenter.fields.source")}</TableHead>
                              <TableHead>{t("admin.configCenter.fields.filename")}</TableHead>
                              <TableHead>{t("admin.configCenter.fields.parseStatus")}</TableHead>
                              <TableHead>{t("admin.configCenter.fields.lastSeenAt")}</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {snapshot.inventories.map((item) => (
                              <TableRow key={item.id}>
                                <TableCell>
                                  <Badge variant="secondary">{t(`admin.configCenter.source.${item.source}`)}</Badge>
                                </TableCell>
                                <TableCell>{item.filename}</TableCell>
                                <TableCell>
                                  <Badge
                                    variant={item.parse_status === "ok" ? "success" : "warning"}
                                  >
                                    {item.parse_status}
                                  </Badge>
                                </TableCell>
                                <TableCell>{formatDateTime(item.last_seen_at)}</TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      )}
                    </div>

                    <div>
                      <h3 className="mb-3 text-sm font-semibold text-foreground">
                        {t("admin.configCenter.snapshot.inboundTitle")}
                      </h3>
                      {!snapshot || snapshot.inbound_indexes.length === 0 ? (
                        <EmptyState
                          icon={<GitCompare className="h-full w-full" />}
                          title={t("admin.configCenter.empty.noInboundIndexTitle")}
                          description={t("admin.configCenter.empty.noInboundIndexDescription")}
                          size="sm"
                        />
                      ) : (
                        <Table aria-label={t("admin.configCenter.snapshot.inboundTitle") as string}>
                          <TableHeader>
                            <TableRow>
                              <TableHead>{t("admin.configCenter.fields.source")}</TableHead>
                              <TableHead>{t("admin.configCenter.fields.tag")}</TableHead>
                              <TableHead>{t("admin.configCenter.fields.protocol")}</TableHead>
                              <TableHead>{t("admin.configCenter.fields.listen")}</TableHead>
                              <TableHead>{t("admin.configCenter.fields.port")}</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {snapshot.inbound_indexes.map((item) => (
                              <TableRow key={item.id}>
                                <TableCell>
                                  <Badge variant="secondary">{t(`admin.configCenter.source.${item.source}`)}</Badge>
                                </TableCell>
                                <TableCell>{item.tag}</TableCell>
                                <TableCell>{item.protocol || "-"}</TableCell>
                                <TableCell>{item.listen || "-"}</TableCell>
                                <TableCell>{item.port ?? "-"}</TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      )}
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      )}

      <Dialog open={isSpecDialogOpen} onOpenChange={setIsSpecDialogOpen}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>
              {selectedSpec
                ? t("admin.configCenter.specs.editTitle")
                : t("admin.configCenter.specs.createTitle")}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.configCenter.fields.agentHost")}</label>
                <Select
                  value={specForm.agent_host_id ? String(specForm.agent_host_id) : undefined}
                  onValueChange={(value) =>
                    setSpecForm((prev) => ({ ...prev, agent_host_id: Number(value) || 0 }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t("admin.configCenter.placeholders.selectHost")} />
                  </SelectTrigger>
                  <SelectContent>
                    {hosts.map((host) => (
                      <SelectItem key={String(host.id)} value={String(host.id)}>
                        {host.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.configCenter.fields.coreType")}</label>
                <Select
                  value={specForm.core_type}
                  onValueChange={(value) =>
                    setSpecForm((prev) => ({ ...prev, core_type: value as CoreTypeOption }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CORE_OPTIONS.map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.configCenter.fields.tag")}</label>
              <Input
                value={specForm.tag}
                onChange={(event) => setSpecForm((prev) => ({ ...prev, tag: event.target.value }))}
                placeholder={t("admin.configCenter.placeholders.specTag")}
              />
            </div>

            <label className="flex items-center gap-2 text-sm">
              <Switch
                checked={specForm.enabled}
                onCheckedChange={(checked) => setSpecForm((prev) => ({ ...prev, enabled: checked }))}
              />
              {t("admin.configCenter.fields.enabled")}
            </label>

            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.configCenter.fields.semanticSpec")}</label>
              <Textarea
                className="min-h-[140px] font-mono text-xs"
                value={specForm.semantic_spec}
                onChange={(event) =>
                  setSpecForm((prev) => ({ ...prev, semantic_spec: event.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.configCenter.fields.coreSpecific")}</label>
              <Textarea
                className="min-h-[120px] font-mono text-xs"
                value={specForm.core_specific}
                onChange={(event) =>
                  setSpecForm((prev) => ({ ...prev, core_specific: event.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.configCenter.fields.changeNote")}</label>
              <Input
                value={specForm.change_note}
                onChange={(event) =>
                  setSpecForm((prev) => ({ ...prev, change_note: event.target.value }))
                }
                placeholder={t("admin.configCenter.placeholders.optional")}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsSpecDialogOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={handleSaveSpec}
              disabled={createSpecMutation.isPending || updateSpecMutation.isPending}
            >
              {createSpecMutation.isPending || updateSpecMutation.isPending
                ? t("common.loading")
                : selectedSpec
                ? t("common.save")
                : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isHistoryDialogOpen} onOpenChange={setIsHistoryDialogOpen}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>
              {t("admin.configCenter.specs.historyTitle")}
              {historyTarget ? ` · ${historyTarget.tag}` : ""}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            {historyQuery.isLoading ? (
              <Loading />
            ) : historyQuery.error ? (
              <p className="text-sm text-destructive">{t("admin.configCenter.messages.historyLoadFailed")}</p>
            ) : historyItems.length === 0 ? (
              <EmptyState
                icon={<History className="h-full w-full" />}
                title={t("admin.configCenter.empty.noHistoryTitle")}
                description={t("admin.configCenter.empty.noHistoryDescription")}
                size="sm"
              />
            ) : (
              <Table aria-label={t("admin.configCenter.specs.historyTitle") as string}>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("admin.configCenter.fields.revision")}</TableHead>
                    <TableHead>{t("admin.configCenter.fields.changeNote")}</TableHead>
                    <TableHead>{t("admin.configCenter.fields.operator")}</TableHead>
                    <TableHead>{t("admin.configCenter.fields.createdAt")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {historyItems.map((item: ConfigCenterSpecRevision) => (
                    <TableRow key={item.id}>
                      <TableCell>{item.revision}</TableCell>
                      <TableCell className="max-w-[380px] truncate text-muted-foreground">
                        {item.change_note || "-"}
                      </TableCell>
                      <TableCell>{item.operator_id || "-"}</TableCell>
                      <TableCell>{formatDateTime(item.created_at)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsHistoryDialogOpen(false)}>
              {t("common.close")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isImportDialogOpen} onOpenChange={setIsImportDialogOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t("admin.configCenter.import.title")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.configCenter.fields.source")}</label>
                <Select
                  value={importForm.source}
                  onValueChange={(value) =>
                    setImportForm((prev) => ({ ...prev, source: value as ImportFormState["source"] }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="legacy">{t("admin.configCenter.source.legacy")}</SelectItem>
                    <SelectItem value="managed">{t("admin.configCenter.source.managed")}</SelectItem>
                    <SelectItem value="merged">{t("admin.configCenter.source.merged")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.configCenter.fields.enabled")}</label>
                <div className="flex h-10 items-center rounded-md border border-border px-3">
                  <Switch
                    checked={importForm.enabled}
                    onCheckedChange={(checked) => setImportForm((prev) => ({ ...prev, enabled: checked }))}
                  />
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.configCenter.fields.filename")}</label>
                <Input
                  value={importForm.filename}
                  onChange={(event) =>
                    setImportForm((prev) => ({ ...prev, filename: event.target.value }))
                  }
                  placeholder={t("admin.configCenter.placeholders.optional")}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.configCenter.fields.tag")}</label>
                <Input
                  value={importForm.tag}
                  onChange={(event) => setImportForm((prev) => ({ ...prev, tag: event.target.value }))}
                  placeholder={t("admin.configCenter.placeholders.optional")}
                />
              </div>
            </div>

            <label className="flex items-center gap-2 text-sm">
              <Switch
                checked={importForm.overwrite_existing}
                onCheckedChange={(checked) =>
                  setImportForm((prev) => ({ ...prev, overwrite_existing: checked }))
                }
              />
              {t("admin.configCenter.import.overwriteExisting")}
            </label>

            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.configCenter.fields.changeNote")}</label>
              <Input
                value={importForm.change_note}
                onChange={(event) =>
                  setImportForm((prev) => ({ ...prev, change_note: event.target.value }))
                }
                placeholder={t("admin.configCenter.placeholders.optional")}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsImportDialogOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleImport} disabled={importMutation.isPending || !selectedHostId}>
              {importMutation.isPending ? t("common.loading") : t("admin.configCenter.actions.import")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
