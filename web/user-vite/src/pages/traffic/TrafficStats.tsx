import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { ArrowDown, ArrowUp, BarChart3 } from "lucide-react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import { fetchTrafficLogs } from "@/api/traffic";
import { Card, CardContent, CardHeader } from "@/components/ui";
import { Loading, ErrorBanner, EmptyState } from "@/components/ui";
import { formatBytes } from "@/lib/format";
import { QUERY_KEYS } from "@/lib/constants";

export default function TrafficStats() {
  const { t } = useTranslation();
  const {
    data: logs,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.TRAFFIC,
    queryFn: fetchTrafficLogs,
  });

  if (isLoading) return <Loading />;
  if (error)
    return <ErrorBanner message={t("error.loadTraffic")} onRetry={refetch} />;

  if (!logs || logs.length === 0) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold">{t("traffic.title")}</h1>
          <p className="text-sm text-muted-foreground">{t("traffic.subtitle")}</p>
        </div>
        <EmptyState icon={<BarChart3 size={48} />} title={t("traffic.noData")} />
      </div>
    );
  }

  const chartData = logs
    .slice()
    .reverse()
    .map((log) => ({
      date: new Date(log.record_at * 1000).toLocaleDateString(),
      upload: log.u,
      download: log.d,
      total: log.u + log.d,
    }));

  const totalUpload = logs.reduce((sum, log) => sum + log.u, 0);
  const totalDownload = logs.reduce((sum, log) => sum + log.d, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{t("traffic.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("traffic.subtitle")}</p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="p-5">
            <div className="flex items-start gap-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-600">
                <ArrowUp className="h-5 w-5" />
              </div>
              <div className="flex-1">
                <p className="text-sm text-muted-foreground">{t("traffic.upload")}</p>
                <p className="text-2xl font-semibold text-foreground">
                  {formatBytes(totalUpload)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-start gap-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <ArrowDown className="h-5 w-5" />
              </div>
              <div className="flex-1">
                <p className="text-sm text-muted-foreground">{t("traffic.download")}</p>
                <p className="text-2xl font-semibold text-foreground">
                  {formatBytes(totalDownload)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-start gap-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                <BarChart3 className="h-5 w-5" />
              </div>
              <div className="flex-1">
                <p className="text-sm text-muted-foreground">{t("traffic.total")}</p>
                <p className="text-2xl font-semibold text-foreground">
                  {formatBytes(totalUpload + totalDownload)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold">{t("traffic.last30Days")}</h3>
        </CardHeader>
        <CardContent className="pt-0">
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
                <XAxis dataKey="date" tick={{ fontSize: 12 }} tickLine={false} />
                <YAxis
                  tickFormatter={(value) => formatBytes(value)}
                  tick={{ fontSize: 12 }}
                  tickLine={false}
                  axisLine={false}
                />
                <Tooltip
                  formatter={(value) => formatBytes(value as number)}
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: "8px",
                    boxShadow: "0 10px 30px -12px rgba(0,0,0,0.2)",
                  }}
                  labelStyle={{ color: "hsl(var(--foreground))" }}
                />
                <Legend />
                <Line
                  type="monotone"
                  dataKey="upload"
                  name={t("traffic.upload")}
                  stroke="hsl(var(--primary))"
                  strokeWidth={2}
                  dot={false}
                />
                <Line
                  type="monotone"
                  dataKey="download"
                  name={t("traffic.download")}
                  stroke="#22c55e"
                  strokeWidth={2}
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
