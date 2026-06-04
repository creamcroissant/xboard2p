import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Globe, Plus, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { AdminPageShell } from "@/components/admin";
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui";
import { QUERY_KEYS } from "@/lib/constants";
import {
  createCDNSite,
  deleteCDNSite,
  deployCDNSite,
  fetchCDNSites,
  updateCDNSite,
} from "@/api/admin/cdn";
import type { CDNSite, CreateCDNSiteRequest } from "@/api/admin/cdn";
import CDNSiteForm from "./CDNSiteForm";

export default function CDNSiteList() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const queryKey = QUERY_KEYS.ADMIN_CDN_SITES;

  const [formOpen, setFormOpen] = useState(false);
  const [editingSite, setEditingSite] = useState<CDNSite | null>(null);

  const { data: fetched, isLoading } = useQuery({
    queryKey,
    queryFn: () => fetchCDNSites(),
  });
  const sites: CDNSite[] = fetched?.sites ?? [];

  const createMutation = useMutation({
    mutationFn: (data: CreateCDNSiteRequest) => createCDNSite(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      setFormOpen(false);
      toast.success(t("admin.cdn.messages.siteCreated"));
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: CreateCDNSiteRequest & { id: number }) => updateCDNSite(data.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      setFormOpen(false);
      toast.success(t("admin.cdn.messages.siteUpdated"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteCDNSite(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      toast.success(t("admin.cdn.messages.siteDeleted"));
    },
  });

  const deployMutation = useMutation({
    mutationFn: (id: number) => deployCDNSite(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      toast.success(t("admin.cdn.messages.deployed"));
    },
  });

  const handleAdd = () => {
    setEditingSite(null);
    setFormOpen(true);
  };

  const handleEdit = (site: CDNSite) => {
    setEditingSite(site);
    setFormOpen(true);
  };

  const handleSubmit = (data: CreateCDNSiteRequest) => {
    if (editingSite) {
      updateMutation.mutate({ ...data, id: editingSite.id });
    } else {
      createMutation.mutate(data);
    }
  };

  const handleDeploy = (id: number) => {
    deployMutation.mutate(id);
  };

  return (
    <AdminPageShell title={t("admin.cdn.title")} data-testid="cdn-site-list-page">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle data-testid="cdn-list-title">{t("admin.cdn.title")}</CardTitle>
            <CardDescription>{t("admin.cdn.description")}</CardDescription>
          </div>
          <Button onClick={handleAdd} data-testid="cdn-add-site-button">
            <Plus className="mr-2 h-4 w-4" />
            {t("admin.cdn.addSite")}
          </Button>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : sites.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-muted-foreground">
              <Globe className="h-10 w-10" />
              <p>{t("admin.cdn.empty")}</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("admin.cdn.name")}</TableHead>
                  <TableHead>{t("admin.cdn.domain")}</TableHead>
                  <TableHead>{t("admin.cdn.provider")}</TableHead>
                  <TableHead>{t("admin.cdn.deployStatus")}</TableHead>
                  <TableHead>{t("admin.cdn.enabled")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sites.map((site) => (
                  <TableRow key={site.id} data-testid={`cdn-site-row-${site.id}`}>
                    <TableCell className="font-medium">{site.name || "-"}</TableCell>
                    <TableCell>{site.domain}</TableCell>
                    <TableCell>{site.origin_type}</TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          site.status === "active"
                            ? "success"
                            : site.status === "error"
                              ? "destructive"
                              : "secondary"
                        }
                      >
                        {site.status || "pending"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {site.enabled ? (
                        <Badge variant="outline" className="text-green-600">
                          {t("common.yes")}
                        </Badge>
                      ) : (
                        <Badge variant="secondary">{t("common.no")}</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEdit(site)}
                          data-testid={`cdn-edit-${site.id}`}
                        >
                          {t("common.edit")}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDeploy(site.id)}
                          data-testid={`cdn-deploy-${site.id}`}
                        >
                          {t("admin.cdn.deploy")}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => deleteMutation.mutate(site.id)}
                          data-testid={`cdn-delete-${site.id}`}
                        >
                          <Trash2 className="h-4 w-4" />
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

      <CDNSiteForm
        open={formOpen}
        onOpenChange={setFormOpen}
        editingSite={editingSite}
        onSubmit={handleSubmit}
        isPending={createMutation.isPending || updateMutation.isPending}
      />
    </AdminPageShell>
  );
}
