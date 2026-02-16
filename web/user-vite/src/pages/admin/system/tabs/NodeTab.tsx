import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  fetchSettings,
  getKey,
  resetKey,
  revealKey,
  saveSettings,
} from "@/api/admin/settings";
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui";
import ErrorBanner from "@/components/ui/ErrorBanner";

const CATEGORY = "node";
const TEMPLATE_INTERVAL_KEY = {
  pull: "server_pull_interval",
  push: "server_push_interval",
};

interface NodeForm {
  communicationKey: string;
  masked: boolean;
  pullInterval: string;
  pushInterval: string;
  deviceLimitMode: string;
}

const defaultForm: NodeForm = {
  communicationKey: "",
  masked: true,
  pullInterval: "60",
  pushInterval: "60",
  deviceLimitMode: "0",
};

export default function NodeTab() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<NodeForm>(defaultForm);
  const [revealOpen, setRevealOpen] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);

  const settingsQueryKey = useMemo(() => [...QUERY_KEYS.ADMIN_SYSTEM, CATEGORY], []);
  const keyQueryKey = useMemo(() => [...QUERY_KEYS.ADMIN_SYSTEM, "key"], []);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: settingsQueryKey,
    queryFn: () => fetchSettings(CATEGORY),
  });

  const keyQuery = useQuery({
    queryKey: keyQueryKey,
    queryFn: () => getKey(),
  });

  useEffect(() => {
    if (!data) return;
    setForm((prev) => ({
      ...prev,
      pullInterval: data[TEMPLATE_INTERVAL_KEY.pull] ?? "60",
      pushInterval: data[TEMPLATE_INTERVAL_KEY.push] ?? "60",
      deviceLimitMode: data.device_limit_mode ?? "0",
    }));
  }, [data]);

  useEffect(() => {
    if (!keyQuery.data) return;
    setForm((prev) => ({
      ...prev,
      communicationKey: keyQuery.data.key,
      masked: keyQuery.data.masked,
    }));
  }, [keyQuery.data]);

  const saveMutation = useMutation({
    mutationFn: (payload: NodeForm) =>
      saveSettings(CATEGORY, {
        [TEMPLATE_INTERVAL_KEY.pull]: payload.pullInterval.trim(),
        [TEMPLATE_INTERVAL_KEY.push]: payload.pushInterval.trim(),
        device_limit_mode: payload.deviceLimitMode.trim(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: settingsQueryKey });
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

  const revealMutation = useMutation({
    mutationFn: revealKey,
    onSuccess: (payload) => {
      setRevealOpen(false);
      queryClient.invalidateQueries({ queryKey: keyQueryKey });
      setForm((prev) => ({
        ...prev,
        communicationKey: payload.key,
        masked: payload.masked,
      }));
      toast.success(t("common.success"), {
        description: t("admin.system.settings.actions.reveal"),
      });
    },
    onError: (err: Error) => {
      toast.error(t("common.error"), {
        description: err.message,
      });
    },
  });

  const resetMutation = useMutation({
    mutationFn: resetKey,
    onSuccess: (payload) => {
      setResetOpen(false);
      queryClient.invalidateQueries({ queryKey: keyQueryKey });
      setForm((prev) => ({
        ...prev,
        communicationKey: payload.key,
        masked: payload.masked,
      }));
      toast.success(t("common.success"), {
        description: t("admin.system.settings.actions.reset"),
      });
    },
    onError: (err: Error) => {
      toast.error(t("common.error"), {
        description: err.message,
      });
    },
  });

  const handleSave = () => {
    const pull = Number(form.pullInterval);
    const push = Number(form.pushInterval);
    if (!Number.isFinite(pull) || pull <= 0 || !Number.isFinite(push) || push <= 0) {
      toast.warning(t("common.error"), {
        description: t("admin.system.settings.tooltips.pullInterval"),
      });
      return;
    }
    saveMutation.mutate(form);
  };

  if (isLoading || keyQuery.isLoading) return <Loading />;

  if (error || keyQuery.error) {
    return (
      <ErrorBanner
        message={t("admin.system.settings.messages.loadError")}
        onRetry={() => {
          refetch();
          keyQuery.refetch();
        }}
      />
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("admin.system.settings.tabs.node")}</CardTitle>
        <CardDescription>{t("admin.system.settings.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6 max-w-2xl">
          <div className="space-y-2">
            <label className="text-sm font-medium">{t("admin.system.settings.fields.communicationKey")}</label>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
              <Input value={form.communicationKey} readOnly />
              <div className="flex flex-wrap gap-2">
                <Button type="button" variant="outline" onClick={() => setRevealOpen(true)}>
                  {t("admin.system.settings.actions.reveal")}
                </Button>
                <Button type="button" variant="destructive" onClick={() => setResetOpen(true)}>
                  {t("admin.system.settings.actions.reset")}
                </Button>
              </div>
            </div>
            <p className="text-xs text-muted-foreground">{t("admin.system.settings.tooltips.communicationKey")}</p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.system.settings.fields.pullInterval")}</label>
              <Input
                type="number"
                min={1}
                value={form.pullInterval}
                onChange={(e) => setForm((prev) => ({ ...prev, pullInterval: e.target.value }))}
                placeholder={t("admin.system.settings.placeholders.pullInterval")}
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.system.settings.fields.pushInterval")}</label>
              <Input
                type="number"
                min={1}
                value={form.pushInterval}
                onChange={(e) => setForm((prev) => ({ ...prev, pushInterval: e.target.value }))}
                placeholder={t("admin.system.settings.placeholders.pushInterval")}
              />
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">{t("admin.system.settings.fields.deviceLimitMode")}</label>
            <Select value={form.deviceLimitMode} onValueChange={(value) => setForm((prev) => ({ ...prev, deviceLimitMode: value }))}>
              <SelectTrigger>
                <SelectValue placeholder={t("admin.system.settings.fields.deviceLimitMode")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">{t("admin.system.settings.tooltips.deviceLimitMode")}</SelectItem>
                <SelectItem value="1">IP Limit</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Button onClick={handleSave} disabled={saveMutation.isPending}>
            {saveMutation.isPending ? t("common.loading") : t("admin.system.settings.actions.save")}
          </Button>
        </div>
      </CardContent>

      <Dialog open={revealOpen} onOpenChange={setRevealOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("common.confirm")}</DialogTitle>
            <DialogDescription>{t("admin.system.settings.messages.revealConfirm")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevealOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={() => revealMutation.mutate()} disabled={revealMutation.isPending}>
              {revealMutation.isPending ? t("common.loading") : t("common.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={resetOpen} onOpenChange={setResetOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("common.confirm")}</DialogTitle>
            <DialogDescription>{t("admin.system.settings.messages.resetConfirm")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setResetOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => resetMutation.mutate()}
              disabled={resetMutation.isPending}
            >
              {resetMutation.isPending ? t("common.loading") : t("admin.system.settings.actions.reset")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
