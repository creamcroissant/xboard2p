import { useState } from "react";
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
import { getSystemStatus } from "@/api/admin";
import { useAuth } from "@/providers/AuthProvider";
import { Badge, Card, CardContent, CardHeader, ResponsiveGrid } from "@/components/ui";
import { Button } from "@/components/ui/button";
import { Loading, ErrorBanner } from "@/components/ui";
import { formatBytes, formatDate, daysUntil, isExpired } from "@/lib/format";
import { QUERY_KEYS } from "@/lib/constants";

const toneStyles = {
  primary: {
    iconBg: "bg-primary/10",
    iconText: "text-primary",
  },
  success: {
    iconBg: "bg-emerald-500/10",
    iconText: "text-emerald-600",
  },
  warning: {
    iconBg: "bg-amber-500/10",
    iconText: "text-amber-600",
  },
  danger: {
    iconBg: "bg-red-500/10",
    iconText: "text-red-600",
  },
};

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

  if (isLoading) return <Loading />;
  if (error)
    return <ErrorBanner message={t("error.loadProfile")} onRetry={refetch} />;
  if (!user) return <ErrorBanner message={t("error.loadProfile")} onRetry={refetch} />;

  const transferUsed = (user.u || 0) + (user.d || 0);
  const transferEnable = user.transfer_enable || 0;
  const usagePercent =
    transferEnable > 0 ? (transferUsed / transferEnable) * 100 : 0;

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

  const usageTone =
    usagePercent > 80 ? "danger" : usagePercent > 50 ? "warning" : "success";

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
    <div className="space-y-6">
      <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{t("nav.dashboard")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("dashboard.welcome")}, {user.email}
          </p>
        </div>
        <Button
          variant="outline"
          className="gap-2"
          onClick={() => refetch()}
        >
          <RefreshCw size={16} />
          {t("common.refresh")}
        </Button>
      </div>

      <ResponsiveGrid minColWidth={320} gap={16}>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-start gap-4">
              <div
                className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                  toneStyles[expired ? "danger" : days <= 7 ? "warning" : "primary"]
                    .iconBg
                } ${
                  toneStyles[expired ? "danger" : days <= 7 ? "warning" : "primary"]
                    .iconText
                }`}
              >
                <Calendar className="h-5 w-5" />
              </div>
              <div className="flex-1">
                <p className="text-sm text-muted-foreground">
                  {t("dashboard.planStatus")}
                </p>
                <p className="text-2xl font-semibold text-foreground">
                  {getPlanStatus()}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {getExpiryHint()}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-start gap-4">
              <div
                className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                  toneStyles[usageTone].iconBg
                } ${toneStyles[usageTone].iconText}`}
              >
                <Database className="h-5 w-5" />
              </div>
              <div className="flex-1">
                <p className="text-sm text-muted-foreground">
                  {t("dashboard.dataUsage")}
                </p>
                <p className="text-xl font-semibold text-foreground">
                  {`${formatBytes(transferUsed)} / ${formatBytes(transferEnable)}`}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {`${usagePercent.toFixed(1)}% ${t("common.used")}`}
                </p>
                <div className="mt-3 h-2 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className={`h-full transition-all ${
                      usageTone === "danger"
                        ? "bg-red-500"
                        : usageTone === "warning"
                          ? "bg-amber-500"
                          : "bg-primary"
                    }`}
                    style={{ width: `${Math.min(usagePercent, 100)}%` }}
                  />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

      </ResponsiveGrid>

      {user.subscribe_url && (
        <Card>
          <CardHeader className="flex flex-col gap-4 pb-0 md:flex-row md:items-center md:justify-between">
            <div className="space-y-1">
              <p className="text-xs uppercase text-muted-foreground">
                {t("dashboard.subscription")}
              </p>
              <div className="flex items-center gap-2 text-lg font-semibold">
                <LinkIcon className="h-5 w-5 text-primary" />
                <span>{t("dashboard.subscribeUrl")}</span>
              </div>
            </div>
            <Button variant="outline" size="sm" onClick={handleCopy}>
              {copied ? t("common.copied") : t("common.copy")}
            </Button>
          </CardHeader>
          <CardContent className="pt-4">
            <div className="rounded-md border border-border bg-muted/50 p-3 text-sm text-foreground">
              <span className="block break-all font-mono">
                {user.subscribe_url}
              </span>
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {t("dashboard.copyHint")}
            </p>
          </CardContent>
        </Card>
      )}

      {isAdmin && systemStatus && (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold">{t("dashboard.systemStats")}</h2>
          <ResponsiveGrid minColWidth={200} gap={16}>
            <Card>
              <CardContent className="flex items-center gap-4 p-4">
                <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Users className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    {t("admin.system.totalUsers")}
                  </p>
                  <p className="text-2xl font-semibold text-foreground">
                    {systemStatus.user_count}
                  </p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="flex items-center gap-4 p-4">
                <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                  <Server className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    {t("admin.system.totalServers")}
                  </p>
                  <p className="text-2xl font-semibold text-foreground">
                    {systemStatus.server_count}
                  </p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="flex items-center gap-4 p-4">
                <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-amber-500/10 text-amber-600">
                  <MonitorDot className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    {t("admin.system.totalAgents")}
                  </p>
                  <p className="text-2xl font-semibold text-foreground">
                    {systemStatus.agent_count}
                  </p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="flex items-center gap-4 p-4">
                <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-600">
                  <Wifi className="h-5 w-5" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">
                    {t("admin.system.onlineAgents")}
                  </p>
                  <p className="text-2xl font-semibold text-foreground">
                    {systemStatus.online_agent_count}
                  </p>
                </div>
              </CardContent>
            </Card>
          </ResponsiveGrid>

          <Card>
            <CardHeader>
              <h3 className="text-lg font-semibold">{t("dashboard.systemInfo")}</h3>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Code className="h-4 w-4" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">
                      {t("admin.system.version")}
                    </p>
                    <Badge variant="secondary">
                      {systemStatus.version || "v1.0.0"}
                    </Badge>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Code className="h-4 w-4" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">
                      {t("admin.system.goVersion")}
                    </p>
                    <Badge variant="default">
                      {systemStatus.go_version || "go1.25"}
                    </Badge>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Clock className="h-4 w-4" />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">
                      {t("admin.system.uptime")}
                    </p>
                    <p className="font-medium text-foreground">
                      {formatUptime(systemStatus.uptime || 0)}
                    </p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
