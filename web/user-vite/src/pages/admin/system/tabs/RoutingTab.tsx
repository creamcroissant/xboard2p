import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { fetchSettings, saveSettings } from "@/api/admin/settings";
import { QUERY_KEYS } from "@/lib/constants";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Loading,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui";
import ErrorBanner from "@/components/ui/ErrorBanner";

const CATEGORY = "routing";

type RoutingRule = {
  id: string;
  priority: number;
  group: string;
  matchType: string;
  matchValue: string;
  action: string;
};

const templateOptions = [
  { value: "xray", label: "Xray" },
  { value: "singbox", label: "Sing-box" },
  { value: "clash-meta", label: "Clash Meta" },
  { value: "surge", label: "Surge" },
  { value: "stash", label: "Stash" },
];

function parseRules(value?: string): RoutingRule[] {
  if (!value) return [];
  try {
    const parsed = JSON.parse(value);
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter(Boolean)
      .map((item, index) => ({
        id: item.id ?? `${Date.now()}-${index}`,
        priority: Number(item.priority ?? index + 1),
        group: String(item.group ?? ""),
        matchType: String(item.matchType ?? item.match_type ?? "domain"),
        matchValue: String(item.matchValue ?? item.match_value ?? ""),
        action: String(item.action ?? "proxy"),
      }));
  } catch {
    return [];
  }
}

function serializeRules(rules: RoutingRule[]): string {
  return JSON.stringify(
    rules.map((rule) => ({
      id: rule.id,
      priority: rule.priority,
      group: rule.group,
      match_type: rule.matchType,
      match_value: rule.matchValue,
      action: rule.action,
    }))
  );
}

export default function RoutingTab() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [defaultTemplate, setDefaultTemplate] = useState("xray");
  const [rules, setRules] = useState<RoutingRule[]>([]);
  const [editingRule, setEditingRule] = useState<RoutingRule | null>(null);

  const queryKey = useMemo(() => [...QUERY_KEYS.ADMIN_SYSTEM, CATEGORY], []);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey,
    queryFn: () => fetchSettings(CATEGORY),
  });

  useEffect(() => {
    if (!data) return;
    setDefaultTemplate(data.default_template ?? "xray");
    setRules(parseRules(data.custom_rules));
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: (payload: { template: string; rules: RoutingRule[] }) =>
      saveSettings(CATEGORY, {
        default_template: payload.template,
        custom_rules: serializeRules(payload.rules),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      toast.success(t("common.success"), {
        description: t("admin.system.settings.messages.saveSuccess"),
      });
    },
    onError: (err: Error) => {
      toast.error(t("common.error"), {
        description: err.message,
      });
    },
  });

  const handleSave = () => {
    saveMutation.mutate({ template: defaultTemplate, rules });
  };

  const handleAddRule = () => {
    setEditingRule({
      id: `${Date.now()}`,
      priority: rules.length + 1,
      group: "default",
      matchType: "domain",
      matchValue: "",
      action: "proxy",
    });
  };

  const handleEditRule = (rule: RoutingRule) => {
    setEditingRule({ ...rule });
  };

  const handleDeleteRule = (ruleId: string) => {
    setRules((prev) => prev.filter((rule) => rule.id !== ruleId));
  };

  const handleRuleChange = (field: keyof RoutingRule, value: string) => {
    setEditingRule((prev) => (prev ? { ...prev, [field]: value } : prev));
  };

  const handleRuleSave = () => {
    if (!editingRule) return;
    setRules((prev) => {
      const exists = prev.find((rule) => rule.id === editingRule.id);
      if (exists) {
        return prev.map((rule) => (rule.id === editingRule.id ? { ...editingRule } : rule));
      }
      return [...prev, editingRule];
    });
    setEditingRule(null);
  };

  if (isLoading) return <Loading />;

  if (error) {
    return (
      <ErrorBanner
        message={t("admin.system.settings.messages.loadError")}
        onRetry={() => refetch()}
      />
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("admin.system.settings.tabs.routing")}</CardTitle>
        <CardDescription>{t("admin.system.settings.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          <div className="space-y-2">
            <label className="text-sm font-medium">
              {t("admin.system.settings.fields.defaultTemplate")}
            </label>
            <Select value={defaultTemplate} onValueChange={setDefaultTemplate}>
              <SelectTrigger>
                <SelectValue placeholder={t("admin.system.settings.fields.defaultTemplate")} />
              </SelectTrigger>
              <SelectContent>
                {templateOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold">
                {t("admin.system.settings.fields.customRules")}
              </h3>
              <Button variant="outline" onClick={handleAddRule}>
                {t("common.create")}
              </Button>
            </div>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Priority</TableHead>
                    <TableHead>Group</TableHead>
                    <TableHead>Match</TableHead>
                    <TableHead>Action</TableHead>
                    <TableHead>{t("common.actions")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rules.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center text-muted-foreground">
                        {t("common.noData")}
                      </TableCell>
                    </TableRow>
                  ) : (
                    rules.map((rule) => (
                      <TableRow key={rule.id}>
                        <TableCell>{rule.priority}</TableCell>
                        <TableCell>{rule.group}</TableCell>
                        <TableCell>
                          {rule.matchType}: {rule.matchValue}
                        </TableCell>
                        <TableCell>{rule.action}</TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-2">
                            <Button size="sm" variant="outline" onClick={() => handleEditRule(rule)}>
                              {t("common.edit")}
                            </Button>
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => handleDeleteRule(rule.id)}
                            >
                              {t("common.delete")}
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>

            {editingRule && (
              <div className="rounded-md border border-border p-4 space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Priority</label>
                    <Input
                      type="number"
                      min={1}
                      value={String(editingRule.priority)}
                      onChange={(e) => handleRuleChange("priority", e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Group</label>
                    <Input
                      value={editingRule.group}
                      onChange={(e) => handleRuleChange("group", e.target.value)}
                    />
                  </div>
                </div>

                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Match Type</label>
                    <Select
                      value={editingRule.matchType}
                      onValueChange={(value) => handleRuleChange("matchType", value)}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Select" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="domain">Domain</SelectItem>
                        <SelectItem value="ip">IP</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Match Value</label>
                    <Input
                      value={editingRule.matchValue}
                      onChange={(e) => handleRuleChange("matchValue", e.target.value)}
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-medium">Action</label>
                  <Select
                    value={editingRule.action}
                    onValueChange={(value) => handleRuleChange("action", value)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="proxy">Proxy</SelectItem>
                      <SelectItem value="direct">Direct</SelectItem>
                      <SelectItem value="reject">Reject</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex flex-wrap gap-2">
                  <Button onClick={handleRuleSave}>{t("common.save")}</Button>
                  <Button variant="outline" onClick={() => setEditingRule(null)}>
                    {t("common.cancel")}
                  </Button>
                </div>
              </div>
            )}
          </div>

          <Button onClick={handleSave} disabled={saveMutation.isPending}>
            {saveMutation.isPending ? t("common.loading") : t("admin.system.settings.actions.save")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
