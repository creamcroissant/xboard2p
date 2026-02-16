import { useTranslation } from "react-i18next";
import { ArrowDown, ArrowUp, Clock, Cpu, HardDrive, MemoryStick } from "lucide-react";
import { AgentStatus, type AgentHost } from "@/types";
import ResourceGauge from "./ResourceGauge";
import { formatBytes } from "@/lib/format";
import { Card, CardContent, CardHeader } from "@/components/ui";
import { Badge } from "@/components/ui/badge";

interface AgentStatusCardProps {
  agent: AgentHost;
  onClick?: () => void;
}

function formatRelativeTime(
  timestamp: number,
  t: (key: string, options?: Record<string, unknown>) => string
): string {
  if (!timestamp) return t("admin.agents.never");

  const now = Math.floor(Date.now() / 1000);
  const diff = now - timestamp;

  if (diff < 0) return t("admin.agents.justNow");
  if (diff < 60) return t("admin.agents.secondsAgo", { count: diff });
  if (diff < 3600) return t("admin.agents.minutesAgo", { count: Math.floor(diff / 60) });
  if (diff < 86400) return t("admin.agents.hoursAgo", { count: Math.floor(diff / 3600) });
  return t("admin.agents.daysAgo", { count: Math.floor(diff / 86400) });
}

function getStatusConfig(status: AgentStatus, t: (key: string) => string) {
  switch (status) {
    case AgentStatus.Online:
      return { variant: "success" as const, label: t("admin.agents.status.online") };
    case AgentStatus.Warning:
      return { variant: "warning" as const, label: t("admin.agents.status.warning") };
    case AgentStatus.Offline:
    default:
      return { variant: "danger" as const, label: t("admin.agents.status.offline") };
  }
}

export default function AgentStatusCard({ agent, onClick }: AgentStatusCardProps) {
  const { t } = useTranslation();
  const statusConfig = getStatusConfig(agent.status, t);

  return (
    <Card
      className="transition-shadow hover:shadow-md"
      onClick={onClick}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      data-testid={onClick ? "admin-agent-card" : undefined}
    >
      <CardHeader className="flex items-start justify-between gap-3 pb-2">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <span
              className={`h-2 w-2 rounded-full ${
                agent.status === AgentStatus.Online
                  ? "bg-emerald-500"
                  : agent.status === AgentStatus.Warning
                  ? "bg-amber-500"
                  : "bg-muted-foreground"
              }`}
            />
            <span className="font-semibold text-foreground">{agent.name}</span>
          </div>
          <span className="text-xs text-muted-foreground">
            {agent.host}:{agent.port}
          </span>
        </div>
        <Badge variant={statusConfig.variant}>{statusConfig.label}</Badge>
      </CardHeader>

      <CardContent className="space-y-3">
        <div className="space-y-2 text-muted-foreground">
          <div className="flex items-center gap-2">
            <Cpu className="h-4 w-4" />
            <ResourceGauge label="CPU" used={agent.cpu_used} total={100} unit="%" showPercentage={false} />
          </div>

          <div className="flex items-center gap-2">
            <MemoryStick className="h-4 w-4" />
            <ResourceGauge
              label={t("admin.agents.memory")}
              used={agent.mem_used}
              total={agent.mem_total}
              unit="bytes"
            />
          </div>

          <div className="flex items-center gap-2">
            <HardDrive className="h-4 w-4" />
            <ResourceGauge
              label={t("admin.agents.disk")}
              used={agent.disk_used}
              total={agent.disk_total}
              unit="bytes"
            />
          </div>
        </div>

        <div className="flex items-center justify-between text-sm">
          <div className="flex items-center gap-1 text-emerald-600">
            <ArrowUp className="h-4 w-4" />
            <span>{formatBytes(agent.upload_total)}</span>
          </div>
          <div className="flex items-center gap-1 text-primary">
            <ArrowDown className="h-4 w-4" />
            <span>{formatBytes(agent.download_total)}</span>
          </div>
        </div>

        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>
            {t("admin.agents.lastHeartbeat")}: {formatRelativeTime(agent.last_heartbeat_at, t)}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

