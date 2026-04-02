import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Plus, RefreshCw, Server } from "lucide-react";
import { QUERY_KEYS } from "@/lib/constants";
import { getAgentHosts, refreshAgentHosts, updateAgentHost } from "@/api/admin";
import { fetchSettings, revealKey } from "@/api/admin/settings";
import { AgentStatusCard } from "@/components/admin";
import { EmptyState, Loading } from "@/components/ui";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Card, CardContent, CardHeader, ResponsiveGrid, Input } from "@/components/ui";
import type { AgentHost, UpdateAgentHostRequest } from "@/types";
import AgentCorePanel from "./AgentCorePanel";

const DEFAULT_DEPLOY_SCRIPT_URL =
  "https://raw.githubusercontent.com/creamcroissant/xboard2p/main/deploy/agent.sh";

function sanitizeShellArgument(value: string): string {
  return value.replace(/[\r\n]/g, "").trim();
}

function shellEscapeSingleQuoted(value: string): string {
  const sanitized = sanitizeShellArgument(value);
  return `'${sanitized.replace(/'/g, `'"'"'`)}'`;
}

function resolveDeployScriptURL(): string {
  const runtimeURL = (window.settings?.deploy_script_url ?? "").trim();
  return runtimeURL || DEFAULT_DEPLOY_SCRIPT_URL;
}

function buildDeployCommand(communicationKey: string, grpcAddress: string): string {
  const deployScriptURL = resolveDeployScriptURL();
  return [
    `curl -fsSL ${shellEscapeSingleQuoted(deployScriptURL)} -o /tmp/agent.sh`,
    `sudo INSTALL_DIR=/opt/xboard/agent sh /tmp/agent.sh -k ${shellEscapeSingleQuoted(communicationKey)} -g ${shellEscapeSingleQuoted(grpcAddress)}`,
  ].join(" && ");
}

function fallbackCopyText(text: string): boolean {
  if (typeof document === "undefined") {
    return false;
  }
  const textArea = document.createElement("textarea");
  textArea.value = text;
  textArea.setAttribute("readonly", "");
  textArea.style.position = "fixed";
  textArea.style.top = "0";
  textArea.style.left = "0";
  textArea.style.opacity = "0";
  document.body.appendChild(textArea);
  textArea.focus();
  textArea.select();
  let copied = false;
  try {
    copied = document.execCommand("copy");
  } catch {
    copied = false;
  }
  document.body.removeChild(textArea);
  return copied;
}

async function copyText(text: string): Promise<boolean> {
  if (!text) return false;
  if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fallback to execCommand for insecure context or permission-denied cases.
    }
  }
  return fallbackCopyText(text);
}

export default function AgentList() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isCorePanelOpen, setIsCorePanelOpen] = useState(false);
  const [isDeployDialogOpen, setIsDeployDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState<AgentHost | null>(null);
  const [deployCommand, setDeployCommand] = useState("");
  const [deployMissingAddress, setDeployMissingAddress] = useState(false);
  const [editForm, setEditForm] = useState({ name: "", host: "" });

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: QUERY_KEYS.ADMIN_AGENTS,
    queryFn: () => getAgentHosts({ page: 1, page_size: 100 }),
    refetchInterval: 30000,
  });

  const deployMutation = useMutation({
    mutationFn: async () => {
      const [nodeSettings, keyInfo] = await Promise.all([fetchSettings("node"), revealKey()]);
      const grpcAddress = (nodeSettings.agent_grpc_address ?? "").trim();
      const communicationKey = (keyInfo.key || "").trim();
      return {
        grpcAddress,
        communicationKey,
      };
    },
    onSuccess: ({ grpcAddress, communicationKey }) => {
      setIsDialogOpen(false);
      const hasGrpcAddress = grpcAddress.length > 0;
      const hasCommunicationKey = communicationKey.length > 0;
      setDeployMissingAddress(!hasGrpcAddress);
      setDeployCommand(
        hasGrpcAddress && hasCommunicationKey
          ? buildDeployCommand(communicationKey, grpcAddress)
          : ""
      );
      setIsDeployDialogOpen(true);
    },
    onError: (err: Error) => {
      toast.error(t("admin.agents.deploy.title"), { description: err.message });
    },
  });

  const refreshMutation = useMutation({
    mutationFn: refreshAgentHosts,
    onSuccess: () => {
      refetch();
      toast.success(t("admin.agents.refreshSuccess"));
    },
  });

  const updateMutation = useMutation({
    mutationFn: (payload: UpdateAgentHostRequest) => updateAgentHost(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENTS });
      setIsEditDialogOpen(false);
      setSelectedAgent(null);
      setEditForm({ name: "", host: "" });
      toast.success(t("admin.agents.updateSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.agents.updateError"), { description: err.message });
    },
  });

  const handleOpenDeployDialog = () => {
    deployMutation.mutate();
  };

  const handleDialogChange = (open: boolean) => {
    setIsDialogOpen(open);
  };

  const handleCorePanelChange = (open: boolean) => {
    setIsCorePanelOpen(open);
    if (!open) {
      setSelectedAgent(null);
    }
  };

  const handleEditDialogChange = (open: boolean) => {
    setIsEditDialogOpen(open);
    if (!open) {
      setSelectedAgent(null);
      setEditForm({ name: "", host: "" });
    }
  };

  const handleDeployDialogChange = (open: boolean) => {
    setIsDeployDialogOpen(open);
    if (!open) {
      setDeployCommand("");
      setDeployMissingAddress(false);
    }
  };

  const handleOpenCorePanel = (agent: AgentHost) => {
    setSelectedAgent(agent);
    setIsCorePanelOpen(true);
  };

  const handleOpenEditDialog = (agent: AgentHost) => {
    setSelectedAgent(agent);
    setEditForm({
      name: agent.name || "",
      host: agent.host || "",
    });
    setIsEditDialogOpen(true);
  };

  const handleSaveEdit = () => {
    if (!selectedAgent) return;
    const name = editForm.name.trim();
    const host = editForm.host.trim();
    if (!name || !host) {
      toast.warning(t("common.error"), {
        description: t("admin.agents.nameHostRequired"),
      });
      return;
    }
    updateMutation.mutate({
      id: selectedAgent.id,
      name,
      host,
    });
  };

  const handleCopyDeployCommand = async () => {
    if (!deployCommand) return;
    const copied = await copyText(deployCommand);
    if (copied) {
      toast.success(t("common.success"), {
        description: t("admin.agents.deploy.copySuccess"),
      });
      return;
    }
    toast.error(t("common.error"), {
      description: t("admin.agents.deploy.copyError"),
    });
  };

  const agents: AgentHost[] = data?.data || [];

  if (isLoading) return <Loading />;

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <p className="text-sm text-destructive">{t("admin.agents.loadError")}</p>
        <Button onClick={() => refetch()}>{t("common.retry")}</Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-2xl font-semibold">{t("admin.agents.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("admin.agents.description", { count: agents.length })}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            onClick={() => refreshMutation.mutate()}
            disabled={refreshMutation.isPending}
          >
            <RefreshCw className="mr-2 h-4 w-4" />
            {refreshMutation.isPending ? t("common.loading") : t("common.refresh")}
          </Button>
          <Button
            data-testid="admin-agents-add-button"
            onClick={() => setIsDialogOpen(true)}
            disabled={deployMutation.isPending}
          >
            <Plus className="mr-2 h-4 w-4" />
            {deployMutation.isPending ? t("common.loading") : t("admin.agents.add")}
          </Button>
        </div>
      </div>

      {agents.length === 0 ? (
        <EmptyState
          icon={<Server className="h-full w-full" />}
          title={t("admin.agents.empty")}
          description={
            t("admin.agents.emptyDescription") ||
            "Add your first agent to start monitoring your nodes"
          }
          action={
            <Button
              data-testid="admin-agents-add-button-empty"
              onClick={() => setIsDialogOpen(true)}
              disabled={deployMutation.isPending}
            >
              <Plus className="mr-2 h-4 w-4" />
              {deployMutation.isPending ? t("common.loading") : t("admin.agents.add")}
            </Button>
          }
          size="lg"
        />
      ) : (
        <ResponsiveGrid minColWidth={280} gap={16}>
          {agents.map((agent) => (
            <AgentStatusCard
              key={agent.id}
              agent={agent}
              onClick={() => handleOpenCorePanel(agent)}
              onEdit={() => handleOpenEditDialog(agent)}
            />
          ))}
        </ResponsiveGrid>
      )}

      <Dialog open={isCorePanelOpen} onOpenChange={handleCorePanelChange}>
        <DialogContent className="sm:max-w-6xl">
          {selectedAgent && (
            <AgentCorePanel agentHostId={selectedAgent.id} agentName={selectedAgent.name} />
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={isEditDialogOpen} onOpenChange={handleEditDialogChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admin.agents.addTitle")}</DialogTitle>
            <DialogDescription>{t("admin.agents.description", { count: agents.length })}</DialogDescription>
          </DialogHeader>
          <Card className="border-none shadow-none">
            <CardHeader className="px-0 pb-2 text-sm font-medium text-muted-foreground">
              {t("admin.agents.addTitle")}
            </CardHeader>
            <CardContent className="space-y-4 px-0">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.agents.name")}</label>
                <Input
                  value={editForm.name}
                  onChange={(event) =>
                    setEditForm((prev) => ({
                      ...prev,
                      name: event.target.value,
                    }))
                  }
                  placeholder={t("admin.agents.namePlaceholder")}
                  className="h-10"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.agents.host")}</label>
                <Input
                  value={editForm.host}
                  onChange={(event) =>
                    setEditForm((prev) => ({
                      ...prev,
                      host: event.target.value,
                    }))
                  }
                  placeholder={t("admin.agents.hostPlaceholder")}
                  className="h-10"
                />
              </div>
            </CardContent>
          </Card>
          <DialogFooter>
            <Button variant="outline" onClick={() => handleEditDialogChange(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleSaveEdit} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? t("common.loading") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isDialogOpen} onOpenChange={handleDialogChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admin.agents.deploy.title")}</DialogTitle>
            <DialogDescription>{t("admin.agents.deploy.description")}</DialogDescription>
          </DialogHeader>
          <Card className="border-none shadow-none">
            <CardHeader className="px-0 pb-2 text-sm font-medium text-muted-foreground">
              {t("admin.agents.deploy.title")}
            </CardHeader>
            <CardContent className="space-y-4 px-0">
              <p className="text-sm text-muted-foreground">{t("admin.agents.deploy.description")}</p>
            </CardContent>
          </Card>
          <DialogFooter>
            <Button variant="outline" onClick={() => handleDialogChange(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleOpenDeployDialog} disabled={deployMutation.isPending}>
              {deployMutation.isPending ? t("common.loading") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isDeployDialogOpen} onOpenChange={handleDeployDialogChange}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>{t("admin.agents.deploy.title")}</DialogTitle>
            <DialogDescription>{t("admin.agents.deploy.description")}</DialogDescription>
          </DialogHeader>
          {deployCommand ? (
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.agents.deploy.command")}</label>
              <pre className="max-h-72 overflow-x-auto whitespace-pre-wrap break-all rounded-md border border-border bg-muted p-3 text-xs font-mono">
                {deployCommand}
              </pre>
            </div>
          ) : (
            <p className="text-sm text-amber-600">
              {deployMissingAddress
                ? t("admin.agents.deploy.missingAddress")
                : t("admin.agents.deploy.missingCommunicationKey")}
            </p>
          )}
          <DialogFooter>
            {deployCommand && (
              <Button type="button" variant="outline" onClick={handleCopyDeployCommand}>
                {t("admin.agents.deploy.copy")}
              </Button>
            )}
            <Button type="button" onClick={() => handleDeployDialogChange(false)}>
              {t("common.close")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
