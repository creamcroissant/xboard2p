import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Server, Tag } from "lucide-react";
import { fetchUserServers } from "@/api/server";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Loading, ErrorBanner, EmptyState } from "@/components/ui";
import { QUERY_KEYS } from "@/lib/constants";

export default function Servers() {
  const { t } = useTranslation();
  const {
    data: servers,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: QUERY_KEYS.SERVERS,
    queryFn: fetchUserServers,
  });

  if (isLoading) return <Loading />;
  if (error)
    return <ErrorBanner message={t("error.loadServers")} onRetry={refetch} />;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{t("servers.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("servers.subtitle")}</p>
      </div>

      {!servers || servers.length === 0 ? (
        <EmptyState
          icon={<Server className="h-full w-full" />}
          title={t("servers.noServers")}
          description={t("servers.noServersHint")}
          size="lg"
        />
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("servers.title")}</TableHead>
                  <TableHead className="w-28">{t("servers.rate")}</TableHead>
                  <TableHead className="w-28">{t("servers.online")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {servers.map((server) => (
                  <TableRow key={server.id}>
                    <TableCell className="py-3">
                      <div className="flex items-start gap-3">
                        <div
                          className={`flex h-9 w-9 items-center justify-center rounded-lg border text-muted-foreground ${
                            server.is_online
                              ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-600"
                              : "border-border bg-muted/50"
                          }`}
                        >
                          <Server className="h-4 w-4" />
                        </div>
                        <div className="min-w-0">
                          <p className="font-medium text-foreground truncate">
                            {server.name}
                          </p>
                          <div className="mt-1 flex flex-wrap items-center gap-2">
                            <Badge variant="secondary">{server.type}</Badge>
                            {server.tags && server.tags.length > 0 && (
                              <div className="flex flex-wrap items-center gap-1">
                                {server.tags.map((tag) => (
                                  <span
                                    key={tag}
                                    className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
                                  >
                                    <Tag className="h-3 w-3" />
                                    {tag}
                                  </span>
                                ))}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      x{server.rate}
                    </TableCell>
                    <TableCell>
                      <Badge variant={server.is_online ? "success" : "danger"}>
                        {server.is_online
                          ? t("servers.online")
                          : t("servers.offline")}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
