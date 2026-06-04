import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import {
  Calendar,
  Database,
  RefreshCw,
  Link as LinkIcon,
  Users,
  Server,
  MonitorDot,
  Wifi,
  Clock,
  Code,
} from "lucide-react";
import { fetchUserInfo } from "@/api/user";
import { getQueueStats, getSystemStatus } from "@/api/admin";
import { useAuth } from "@/providers/AuthProvider";
import { Badge, Card, CardContent, CardHeader, ResponsiveGrid } from "@/components/ui";
import StatCard from "@/components/ui/StatCard";
import { Button } from "@/components/ui/button";
import { Loading, ErrorBanner } from "@/components/ui";
import { formatBytes, formatDate, formatDateTime, daysUntil, isExpired } from "@/lib/format";
import { QUERY_KEYS } from "@/lib/constants";

function SectionHeader({
  title,
  description,
  action,
}: {
  title: string;
  description?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
      <div className="space-y-1">
        <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
        {description && <p className="text-sm text-muted-foreground">{description}</p>}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  );
}

function DetailItem({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: ReactNode;
}) {
  return (
    <div className="flex items-center gap-3 rounded-md border bg-card p-3.5">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
        {icon}
      </div>
      <div className="min-w-0">
        <p className="text-sm text-muted-foreground">{label}</p>
        <div className="truncate text-sm font-semibold text-foreground">{value}</div>
      </div>
    </div>
  );
}

function MetricTile({
  label,
  value,
}: {
  label: string;
  value: ReactNode;
}) {
  return (
    <div className="rounded-md border bg-muted/30 p-3.5">
      <p className="text-sm text-muted-foreground">{label}</p>
      <div className="mt-1 break-words text-xl font-semibold leading-tight text-foreground">{value}</div>
    </div>
  );
}

export default function Dashboard() {
  const { t } = useTranslation();
  const { isAdmin } = useAuth();
  const [copied, setCopied] = useState(false);
  const {
    data: user,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.USER_INFO,
    queryFn: fetchUserInfo,
  });
  const { data: systemStatus } = useQuery({
    queryKey: QUERY_KEYS.ADMIN_SYSTEM,
    queryFn: getSystemStatus,
    enabled: isAdmin,
    refetchInterval: 60000,
  });
  const { data: queueStats } = useQuery({
    queryKey: QUERY_KEYS.ADMIN_SYSTEM_QUEUE,
    queryFn: getQueueStats,
    enabled: isAdmin,
    refetchInterval: 60000,
  });

  if (isLoading) return <Loading />;
  if (error) return <ErrorBanner message={t("error.loadProfile")} onRetry={refetch} />;
  if (!user) return <ErrorBanner message={t("error.loadProfile")} onRetry={refetch} />;

  const transferUsed = (user.u || 0) + (user.d || 0);
  const transferEnable = user.transfer_enable || 0;
  const usagePercent = transferEnable > 0 ? (transferUsed / transferEnable) * 100 : 0;

  const expired = user.expired_at ? isExpired(user.expired_at) : false;
  const days = user.expired_at ? daysUntil(user.expired_at) : Infinity;

  const getPlanStatus = () => {
    if (!user.plan_id) return t("dashboard.noPlan");
    if (expired) return t("dashboard.expired");
    return t("dashboard.active");
  };

  const getExpiryHint = () => {
    if (!user.expired_at) return t("dashboard.never");
    if (expired) return t("dashboard.expired");
    if (days <= 7) return `${days} ${t("dashboard.daysLeft")}`;
    return formatDate(user.expired_at);
  };

  const usageTone = usagePercent > 80 ? "danger" : usagePercent > 50 ? "warning" : "success";

  const formatUptime = (seconds: number): string => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    const parts = [] as string[];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);

    return parts.length > 0 ? parts.join(" ") : "< 1m";
  };

  const formatStartedAt = (value?: string): string => {
    if (!value) return "-";
    const timestamp = Date.parse(value);
    if (Number.isNaN(timestamp)) return value;
    return formatDateTime(Math.floor(timestamp / 1000));
  };

  const handleCopy = async () => {
    if (!user.subscribe_url) return;
    try {
      await navigator.clipboard.writeText(user.subscribe_url);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div className="space-y-6 lg:space-y-8">
      <header className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="space-y-1.5">
          <h1 className="text-2xl font-semibold tracking-tight">{t("nav.dashboard")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("dashboard.welcome")}, {user.email}
          </p>
        </div>
        <Button variant="outline" className="gap-2 self-start" onClick={() => refetch()}>
          <RefreshCw size={16} />
          {t("common.refresh")}
        </Button>
      </header>

      <section className="space-y-4">
        <ResponsiveGrid minColWidth={300} gap={16}>
          <StatCard
            className="shadow-none"
            title={t("dashboard.planStatus")}
            value={getPlanStatus()}
            hint={getExpiryHint()}
            icon={<Calendar className="h-5 w-5" />}
            variant={expired ? "danger" : days <= 7 ? "warning" : "primary"}
          />

          <StatCard
            className="shadow-none"
            title={t("dashboard.dataUsage")}
            value={`${formatBytes(transferUsed)} / ${formatBytes(transferEnable)}`}
            hint={`${usagePercent.toFixed(1)}% ${t("common.used")}`}
            icon={<Database className="h-5 w-5" />}
            variant={usageTone}
          >
            <div className="h-2.5 w-full overflow-hidden rounded-full bg-muted/80">
              <div
                className={`h-full transition-all ${
                  usageTone === "danger" ? "bg-red-500" : usageTone === "warning" ? "bg-amber-500" : "bg-primary"
                }`}
                style={{ width: `${Math.min(usagePercent, 100)}%` }}
              />
            </div>
          </StatCard>
        </ResponsiveGrid>

        {user.subscribe_url && (
          <Card className="shadow-none">
            <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
              <div className="space-y-1.5">
                <p className="text-xs uppercase tracking-[0.08em] text-muted-foreground">{t("dashboard.subscription")}</p>
                <div className="flex items-center gap-2 text-base font-semibold tracking-tight">
                  <LinkIcon className="h-4 w-4 text-primary" />
                  <span>{t("dashboard.subscribeUrl")}</span>
                </div>
              </div>
              <Button variant="outline" size="sm" onClick={handleCopy}>
                {copied ? t("common.copied") : t("common.copy")}
              </Button>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="rounded-md border bg-muted/30 p-3 text-sm text-foreground">
                <span className="block break-all font-mono leading-6">{user.subscribe_url}</span>
              </div>
              <p className="text-sm text-muted-foreground">{t("dashboard.copyHint")}</p>
            </CardContent>
          </Card>
        )}
      </section>

      {isAdmin && systemStatus && (
        <section className="space-y-5 border-t pt-5 lg:pt-6">
          <SectionHeader title={t("dashboard.systemStats")} description={t("dashboard.systemStatsHint")} />

          <ResponsiveGrid minColWidth={220} gap={14}>
            <StatCard className="shadow-none" title={t("admin.system.totalUsers")} value={systemStatus.user_count} icon={<Users className="h-5 w-5" />} variant="primary" />
            <StatCard className="shadow-none" title={t("admin.system.totalServers")} value={systemStatus.server_count} icon={<Server className="h-5 w-5" />} variant="default" />
            <StatCard className="shadow-none" title={t("admin.system.totalAgents")} value={systemStatus.agent_count} icon={<MonitorDot className="h-5 w-5" />} variant="warning" />
            <StatCard className="shadow-none" title={t("admin.system.onlineAgents")} value={systemStatus.online_agent_count} icon={<Wifi className="h-5 w-5" />} variant="success" />
          </ResponsiveGrid>

          <div className="grid gap-4 xl:grid-cols-3">
            <Card className="shadow-none xl:col-span-2">
              <CardHeader>
                <h3 className="text-lg font-semibold tracking-tight">{t("dashboard.systemInfo")}</h3>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                  <DetailItem icon={<Code className="h-4 w-4" />} label={t("admin.system.version")} value={<Badge variant="secondary">{systemStatus.version || "go-dev"}</Badge>} />
                  <DetailItem icon={<Code className="h-4 w-4" />} label={t("admin.system.goVersion")} value={<Badge variant="default">{systemStatus.go_version || "go1.25"}</Badge>} />
                  <DetailItem icon={<Clock className="h-4 w-4" />} label={t("admin.system.uptime")} value={formatUptime(systemStatus.uptime || 0)} />
                  <DetailItem icon={<Server className="h-4 w-4" />} label={t("dashboard.environment")} value={systemStatus.environment || "-"} />
                  <DetailItem icon={<Server className="h-4 w-4" />} label={t("dashboard.hostname")} value={systemStatus.hostname || "-"} />
                  <DetailItem icon={<Clock className="h-4 w-4" />} label={t("dashboard.startedAt")} value={formatStartedAt(systemStatus.started_at)} />
                </div>
              </CardContent>
            </Card>

            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
              <Card className="shadow-none">
                <CardHeader>
                  <h3 className="text-base font-semibold">{t("dashboard.logs")}</h3>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-3">
                    <MetricTile label={t("dashboard.logInfo")} value={systemStatus.logs?.info ?? 0} />
                    <MetricTile label={t("dashboard.logWarning")} value={systemStatus.logs?.warning ?? 0} />
                    <MetricTile label={t("dashboard.logError")} value={systemStatus.logs?.error ?? 0} />
                    <MetricTile label={t("dashboard.logTotal")} value={systemStatus.logs?.total ?? 0} />
                  </div>
                </CardContent>
              </Card>

              <Card className="shadow-none">
                <CardHeader>
                  <h3 className="text-base font-semibold">{t("dashboard.queueStats")}</h3>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-3">
                    <MetricTile label={t("dashboard.recentJobs")} value={queueStats?.recentJobs ?? 0} />
                    <MetricTile label={t("dashboard.jobsPerMinute")} value={(queueStats?.jobsPerMinute ?? 0).toFixed(1)} />
                    <MetricTile label={t("dashboard.failedJobs")} value={queueStats?.failedJobs ?? 0} />
                    <MetricTile
                      label={t("dashboard.maxThroughputQueue")}
                      value={
                        queueStats?.queueWithMaxThroughput?.name
                          ? `${queueStats.queueWithMaxThroughput.name} (${queueStats.queueWithMaxThroughput.throughput})`
                          : "-"
                      }
                    />
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </section>
      )}
    </div>
  );
}
