import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { BookOpen, FileText } from "lucide-react";
import { fetchKnowledgeArticles } from "@/api/knowledge";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader } from "@/components/ui";
import { Loading, ErrorBanner, EmptyState } from "@/components/ui";
import { QUERY_KEYS } from "@/lib/constants";

export default function Knowledge() {
  const { t, i18n } = useTranslation();
  const {
    data: groupedArticles,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: [...QUERY_KEYS.KNOWLEDGE, i18n.language],
    queryFn: () => fetchKnowledgeArticles(i18n.language),
    refetchInterval: 30000, // 30 seconds
    refetchOnWindowFocus: true,
    staleTime: 10000, // 10 seconds
  });

  const articles = useMemo(() => {
    if (!groupedArticles) {
      return [];
    }
    return Object.values(groupedArticles).flat();
  }, [groupedArticles]);

  if (isLoading) return <Loading />;
  if (error)
    return <ErrorBanner message={t("error.loadKnowledge")} onRetry={refetch} />;

  if (!articles || articles.length === 0) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold">{t("knowledge.title")}</h1>
          <p className="text-sm text-muted-foreground">{t("knowledge.subtitle")}</p>
        </div>
        <EmptyState
          icon={<BookOpen size={48} />}
          title={t("knowledge.noArticles")}
        />
      </div>
    );
  }

  const categories = Array.from(
    new Set(articles.map((article) => article.category || "Uncategorized"))
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{t("knowledge.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("knowledge.subtitle")}</p>
      </div>

      <div className="flex flex-wrap gap-2">
        {categories.map((category) => (
          <Badge key={category} variant="secondary">
            {category}
          </Badge>
        ))}
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        {articles.map((article) => (
          <Card key={article.id} className="transition-shadow hover:shadow-md">
            <CardHeader className="flex items-start gap-3 pb-2">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <FileText size={18} />
              </div>
              <div className="flex-1">
                <h3 className="text-base font-semibold text-foreground">
                  {article.title}
                </h3>
                <div className="mt-1 flex flex-wrap items-center gap-2">
                  {article.category && (
                    <Badge variant="secondary">{article.category}</Badge>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent className="pt-0">
              <p className="text-sm text-muted-foreground line-clamp-2">
                {(article.body ?? "").replace(/<[^>]*>/g, "").slice(0, 150)}...
              </p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
