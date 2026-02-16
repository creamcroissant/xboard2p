import { BookOpen } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { MoreVertical, Pencil, Trash2, Eye, EyeOff, Plus, ArrowUpDown, RefreshCw } from "lucide-react";
import { QUERY_KEYS } from "@/lib/constants";
import {
  deleteKnowledgeArticle,
  getKnowledgeCategories,
  getKnowledgeDetail,
  getKnowledgeList,
  saveKnowledgeArticle,
  sortKnowledgeArticles,
  toggleKnowledgeVisibility,
} from "@/api/admin";
import type { AdminKnowledgeSaveRequest } from "@/types";
import {
  Badge,
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  EmptyState,
  Loading,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui";
import KnowledgeForm from "./KnowledgeForm";

const defaultLanguages = ["zh-CN", "en-US"] as const;

type SortDirection = "asc" | "desc";

type KnowledgeSort = {
  key: "updated_at" | "title" | "language" | "category";
  direction: SortDirection;
};

export default function KnowledgeList() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [sort, setSort] = useState<KnowledgeSort>({ key: "updated_at", direction: "desc" });

  const listQuery = useQuery({
    queryKey: QUERY_KEYS.ADMIN_KNOWLEDGE,
    queryFn: getKnowledgeList,
  });

  const categoriesQuery = useQuery({
    queryKey: [...QUERY_KEYS.ADMIN_KNOWLEDGE, "categories"],
    queryFn: getKnowledgeCategories,
  });

  const detailQuery = useQuery({
    queryKey: [...QUERY_KEYS.ADMIN_KNOWLEDGE, "detail", editingId],
    queryFn: () => getKnowledgeDetail(editingId as number),
    enabled: isFormOpen && Boolean(editingId),
  });

  const saveMutation = useMutation({
    mutationFn: saveKnowledgeArticle,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_KNOWLEDGE });
      handleCloseForm();
      toast.success(t("admin.knowledge.saveSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.knowledge.saveError"), { description: err.message });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteKnowledgeArticle,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_KNOWLEDGE });
      toast.success(t("admin.knowledge.deleteSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.knowledge.deleteError"), { description: err.message });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: toggleKnowledgeVisibility,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_KNOWLEDGE });
      toast.success(t("admin.knowledge.toggleSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.knowledge.toggleError"), { description: err.message });
    },
  });

  const sortMutation = useMutation({
    mutationFn: sortKnowledgeArticles,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_KNOWLEDGE });
      toast.success(t("admin.knowledge.sortSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.knowledge.sortError"), { description: err.message });
    },
  });

  const handleOpenCreate = () => {
    setEditingId(null);
    setIsFormOpen(true);
  };

  const handleOpenEdit = (id: number) => {
    setEditingId(id);
    setIsFormOpen(true);
  };

  const handleCloseForm = () => {
    setIsFormOpen(false);
    setEditingId(null);
  };

  const handleSubmit = (payload: AdminKnowledgeSaveRequest) => {
    saveMutation.mutate(payload);
  };

  const handleDelete = (id: number) => {
    deleteMutation.mutate(id);
  };

  const handleToggle = (id: number) => {
    toggleMutation.mutate(id);
  };

  const handleSortChange = (key: KnowledgeSort["key"]) => {
    setSort((prev) => {
      if (prev.key === key) {
        return { key, direction: prev.direction === "asc" ? "desc" : "asc" };
      }
      return { key, direction: "asc" };
    });
  };

  const handleApplySortOrder = () => {
    if (knowledgeList.length === 0) {
      return;
    }
    const ids = sortedKnowledge.map((item) => item.id);
    sortMutation.mutate(ids);
  };

  const knowledgeList = listQuery.data ?? [];

  const sortedKnowledge = useMemo(() => {
    const items = [...knowledgeList];
    const dir = sort.direction === "asc" ? 1 : -1;
    items.sort((a, b) => {
      const aValue = a[sort.key] ?? "";
      const bValue = b[sort.key] ?? "";
      if (typeof aValue === "number" && typeof bValue === "number") {
        return (aValue - bValue) * dir;
      }
      return String(aValue).localeCompare(String(bValue)) * dir;
    });
    return items;
  }, [knowledgeList, sort]);

  const languages = useMemo(() => {
    const fromList = knowledgeList.map((item) => item.language).filter(Boolean);
    return Array.from(new Set([...defaultLanguages, ...fromList]));
  }, [knowledgeList]);

  if (listQuery.isLoading) {
    return <Loading />;
  }

  if (listQuery.error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <p className="text-sm text-destructive">{t("admin.knowledge.loadError")}</p>
        <Button variant="outline" onClick={() => listQuery.refetch()}>
          {t("common.retry")}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{t("admin.knowledge.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("admin.knowledge.total", { count: knowledgeList.length })}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => listQuery.refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {t("common.refresh")}
          </Button>
          <Button data-testid="admin-knowledge-add-button" onClick={handleOpenCreate}>
            <Plus className="mr-2 h-4 w-4" />
            {t("admin.knowledge.add")}
          </Button>
        </div>
      </div>

      {knowledgeList.length === 0 ? (
        <EmptyState
          icon={<BookOpen size={48} />}
          title={t("admin.knowledge.empty")}
          description={t("admin.knowledge.emptyDescription")}
          action={
            <Button data-testid="admin-knowledge-add-button-empty" onClick={handleOpenCreate}>
              <Plus className="mr-2 h-4 w-4" />
              {t("admin.knowledge.add")}
            </Button>
          }
        />
      ) : (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs text-muted-foreground">
              {t("admin.knowledge.sortHint")} {t(`admin.knowledge.sortLabel.${sort.key}`)}
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={handleApplySortOrder}
              disabled={sortMutation.isPending}
            >
              <ArrowUpDown className="mr-2 h-4 w-4" />
              {t("admin.knowledge.sortApply")}
            </Button>
          </div>

          <Table aria-label={t("admin.knowledge.title")}>
            <TableHeader>
              <TableRow>
                <TableHead>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1"
                    onClick={() => handleSortChange("title")}
                  >
                    {t("admin.knowledge.titleCol")}
                    <ArrowUpDown className="h-3 w-3" />
                  </button>
                </TableHead>
                <TableHead>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1"
                    onClick={() => handleSortChange("language")}
                  >
                    {t("admin.knowledge.language")}
                    <ArrowUpDown className="h-3 w-3" />
                  </button>
                </TableHead>
                <TableHead>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1"
                    onClick={() => handleSortChange("category")}
                  >
                    {t("admin.knowledge.category")}
                    <ArrowUpDown className="h-3 w-3" />
                  </button>
                </TableHead>
                <TableHead>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1"
                    onClick={() => handleSortChange("updated_at")}
                  >
                    {t("admin.knowledge.updatedAt")}
                    <ArrowUpDown className="h-3 w-3" />
                  </button>
                </TableHead>
                <TableHead>{t("admin.knowledge.status")}</TableHead>
                <TableHead>{t("common.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedKnowledge.map((item) => (
                <TableRow key={item.id}>
                  <TableCell
                    className="font-medium max-w-xs truncate"
                    title={item.title}
                    data-testid="admin-knowledge-title"
                  >
                    {item.title}
                  </TableCell>
                  <TableCell>{item.language}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{item.category}</Badge>
                  </TableCell>
                  <TableCell>{new Date(item.updated_at * 1000).toLocaleString()}</TableCell>
                  <TableCell>
                    <Badge variant={item.show ? "success" : "default"}>
                      {item.show ? t("admin.knowledge.visible") : t("admin.knowledge.hidden")}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" aria-label={t("common.actions")}>
                          <MoreVertical className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem className="gap-2" onSelect={() => handleToggle(item.id)}>
                          {item.show ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                          {item.show ? t("admin.knowledge.hide") : t("admin.knowledge.show")}
                        </DropdownMenuItem>
                        <DropdownMenuItem className="gap-2" onSelect={() => handleOpenEdit(item.id)}>
                          <Pencil className="h-4 w-4" />
                          {t("common.edit")}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="gap-2 text-red-600 focus:text-red-600"
                          onSelect={() => handleDelete(item.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                          {t("common.delete")}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <KnowledgeForm
        open={isFormOpen}
        categories={categoriesQuery.data ?? []}
        languages={languages}
        initialData={detailQuery.data ?? null}
        onSubmit={handleSubmit}
        onCancel={handleCloseForm}
        isSubmitting={saveMutation.isPending}
      />
    </div>
  );
}
