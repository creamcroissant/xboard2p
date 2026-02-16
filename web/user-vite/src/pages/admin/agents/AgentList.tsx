import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Plus, RefreshCw, Server } from "lucide-react";
import { QUERY_KEYS } from "@/lib/constants";
import { getAgentHosts, createAgentHost, refreshAgentHosts } from "@/api/admin";
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
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, ResponsiveGrid } from "@/components/ui";
import type { AgentHost, CreateAgentHostRequest } from "@/types";
import AgentCorePanel from "./AgentCorePanel";

export default function AgentList() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isCorePanelOpen, setIsCorePanelOpen] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState<AgentHost | null>(null);
  const [newAgent, setNewAgent] = useState<CreateAgentHostRequest>({
    name: "",
    host: "",
    port: 9527,
  });

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: QUERY_KEYS.ADMIN_AGENTS,
    queryFn: () => getAgentHosts({ page: 1, page_size: 100 }),
    refetchInterval: 30000,
  });

  const createMutation = useMutation({
    mutationFn: createAgentHost,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_AGENTS });
      setIsDialogOpen(false);
      setNewAgent({ name: "", host: "", port: 9527 });
      toast.success(t("admin.agents.createSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.agents.createError"), { description: err.message });
    },
  });

  const refreshMutation = useMutation({
    mutationFn: refreshAgentHosts,
    onSuccess: () => {
      refetch();
      toast.success(t("admin.agents.refreshSuccess"));
    },
  });

  const handleCreate = () => {
    if (!newAgent.name || !newAgent.host) {
      toast.warning(t("admin.agents.validationError"), {
        description: t("admin.agents.nameHostRequired"),
      });
      return;
    }
    createMutation.mutate(newAgent);
  };

  const handleDialogChange = (open: boolean) => {
    setIsDialogOpen(open);
    if (!open) {
      setNewAgent({ name: "", host: "", port: 9527 });
    }
  };

  const handleCorePanelChange = (open: boolean) => {
    setIsCorePanelOpen(open);
    if (!open) {
      setSelectedAgent(null);
    }
  };

  const handleOpenCorePanel = (agent: AgentHost) => {
    setSelectedAgent(agent);
    setIsCorePanelOpen(true);
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
          <Button data-testid="admin-agents-add-button" onClick={() => setIsDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("admin.agents.add")}
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
            <Button data-testid="admin-agents-add-button-empty" onClick={() => setIsDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t("admin.agents.add")}
            </Button>
          }
          size="lg"
        />
      ) : (
        <ResponsiveGrid minColWidth={280} gap={16}>
          {agents.map((agent) => (
            <AgentStatusCard key={agent.id} agent={agent} onClick={() => handleOpenCorePanel(agent)} />
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

      <Dialog open={isDialogOpen} onOpenChange={handleDialogChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admin.agents.addTitle")}</DialogTitle>
            <DialogDescription>
              {t("admin.agents.description", { count: agents.length })}
            </DialogDescription>
          </DialogHeader>
          <Card className="border-none shadow-none">
            <CardHeader className="px-0 pb-2 text-sm font-medium text-muted-foreground">
              {t("admin.agents.addTitle")}
            </CardHeader>
            <CardContent className="space-y-4 px-0">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.agents.name")}</label>
                <Input
                  value={newAgent.name}
                  onChange={(event) =>
                    setNewAgent({ ...newAgent, name: event.target.value })
                  }
                  placeholder={t("admin.agents.namePlaceholder")}
                  required
                  className="h-10"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.agents.host")}</label>
                <Input
                  value={newAgent.host}
                  onChange={(event) =>
                    setNewAgent({ ...newAgent, host: event.target.value })
                  }
                  placeholder={t("admin.agents.hostPlaceholder")}
                  required
                  className="h-10"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.agents.port")}</label>
                <Input
                  type="number"
                  value={String(newAgent.port)}
                  onChange={(event) =>
                    setNewAgent({
                      ...newAgent,
                      port: parseInt(event.target.value, 10) || 9527,
                    })
                  }
                  placeholder="9527"
                  className="h-10"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t("admin.agents.token")}</label>
                <Input
                  value={newAgent.token || ""}
                  onChange={(event) =>
                    setNewAgent({ ...newAgent, token: event.target.value })
                  }
                  placeholder={t("admin.agents.tokenPlaceholder")}
                  className="h-10"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.agents.tokenDescription")}
                </p>
              </div>
            </CardContent>
          </Card>
          <DialogFooter>
            <Button variant="outline" onClick={() => handleDialogChange(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleCreate} disabled={createMutation.isPending}>
              {createMutation.isPending ? t("common.loading") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
