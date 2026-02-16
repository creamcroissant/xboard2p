import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { CreditCard, Zap, Smartphone, Database, Calendar } from "lucide-react";
import { fetchUserInfo } from "@/api/user";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader } from "@/components/ui";
import { Loading, ErrorBanner, EmptyState } from "@/components/ui";
import { formatBytes, formatDate } from "@/lib/format";
import { QUERY_KEYS } from "@/lib/constants";

export default function Plans() {
  const { t } = useTranslation();
  const {
    data: user,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.USER_INFO,
    queryFn: fetchUserInfo,
  });

  if (isLoading) return <Loading />;
  if (error)
    return <ErrorBanner message={t("error.loadPlans")} onRetry={refetch} />;

  if (!user?.plan) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold">{t("plans.title")}</h1>
          <p className="text-sm text-muted-foreground">{t("plans.subtitle")}</p>
        </div>
        <EmptyState icon={<CreditCard size={48} />} title={t("plans.noPlan")} />
      </div>
    );
  }

  const plan = user.plan;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{t("plans.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("plans.subtitle")}</p>
      </div>

      <Card>
        <CardHeader className="flex flex-col gap-4 pb-0 md:flex-row md:items-center md:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
              <CreditCard className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-xl font-semibold">{plan.name}</h3>
              <Badge variant="success" className="mt-1">
                {t("plans.currentPlan")}
              </Badge>
            </div>
          </div>
        </CardHeader>

        <CardContent className="pt-6">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <div className="flex items-start gap-3 rounded-lg border border-border/60 bg-muted/30 p-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                <Database className="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t("plans.traffic")}</p>
                <p className="mt-1 text-sm font-semibold">
                  {formatBytes(plan.transfer_enable)}
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 rounded-lg border border-border/60 bg-muted/30 p-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                <Zap className="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t("plans.speedLimit")}</p>
                <p className="mt-1 text-sm font-semibold">
                  {plan.speed_limit ? `${plan.speed_limit} Mbps` : t("plans.unlimited")}
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 rounded-lg border border-border/60 bg-muted/30 p-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                <Smartphone className="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t("plans.deviceLimit")}</p>
                <p className="mt-1 text-sm font-semibold">
                  {plan.device_limit || t("plans.unlimited")}
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 rounded-lg border border-border/60 bg-muted/30 p-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-muted">
                <Calendar className="h-4 w-4 text-muted-foreground" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{t("plans.expiresAt")}</p>
                <p className="mt-1 text-sm font-semibold">
                  {user.expired_at ? formatDate(user.expired_at) : t("dashboard.never")}
                </p>
              </div>
            </div>
          </div>

          {plan.content && (
            <div className="mt-6 border-t border-border/60 pt-4">
              <div
                className="prose prose-sm max-w-none text-foreground dark:prose-invert"
                dangerouslySetInnerHTML={{ __html: plan.content }}
              />
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
