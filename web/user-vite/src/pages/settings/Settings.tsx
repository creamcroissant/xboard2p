import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Eye, EyeOff, Key, Lock, RefreshCw } from "lucide-react";
import { changePassword, resetSubscribeToken } from "@/api/user";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader } from "@/components/ui";
import { QUERY_KEYS } from "@/lib/constants";

export default function Settings() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPasswords, setShowPasswords] = useState(false);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const changePasswordMutation = useMutation({
    mutationFn: () => changePassword(currentPassword, newPassword),
    onSuccess: () => {
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.success(t("common.success"), {
        description: t("settings.passwordChanged"),
      });
    },
    onError: (error) => {
      toast.error(t("common.error"), {
        description: error.message,
      });
    },
  });

  const resetTokenMutation = useMutation({
    mutationFn: resetSubscribeToken,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.USER_INFO });
      setIsDialogOpen(false);
      toast.success(t("common.success"), {
        description: t("settings.tokenReset"),
      });
    },
    onError: (error) => {
      toast.error(t("common.error"), {
        description: error.message,
      });
    },
  });

  const handleChangePassword = (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) return;
    changePasswordMutation.mutate();
  };

  const passwordMismatch = confirmPassword && newPassword !== confirmPassword;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{t("settings.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("settings.subtitle")}</p>
      </div>

      <Card>
        <CardHeader className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Lock className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-semibold">
              {t("settings.changePassword")}
            </h3>
            <p className="text-sm text-muted-foreground">{t("settings.security")}</p>
          </div>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleChangePassword} className="space-y-4 max-w-md">
            <div className="space-y-2">
              <label className="text-sm font-medium">
                {t("settings.currentPassword")}
              </label>
              <div className="relative">
                <Input
                  type={showPasswords ? "text" : "password"}
                  value={currentPassword}
                  onChange={(event) => setCurrentPassword(event.target.value)}
                  required
                  className="h-11 pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPasswords((value) => !value)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  {showPasswords ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">
                {t("settings.newPassword")}
              </label>
              <Input
                type={showPasswords ? "text" : "password"}
                value={newPassword}
                onChange={(event) => setNewPassword(event.target.value)}
                required
                className="h-11"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">
                {t("settings.confirmPassword")}
              </label>
              <Input
                type={showPasswords ? "text" : "password"}
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                required
                className="h-11"
              />
              {passwordMismatch && (
                <p className="text-sm text-destructive">
                  {t("settings.passwordMismatch")}
                </p>
              )}
            </div>

            {changePasswordMutation.error && (
              <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
                {changePasswordMutation.error.message}
              </div>
            )}

            <Button
              type="submit"
              disabled={!!passwordMismatch || changePasswordMutation.isPending}
            >
              {changePasswordMutation.isPending
                ? t("common.loading")
                : t("common.save")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-500/10 text-amber-600">
            <Key className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-semibold">
              {t("settings.resetToken")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("settings.resetTokenHint")}
            </p>
          </div>
        </CardHeader>
        <CardContent>
          <Button variant="outline" onClick={() => setIsDialogOpen(true)}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {t("settings.resetToken")}
          </Button>
        </CardContent>
      </Card>

      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("common.confirm")}</DialogTitle>
            <DialogDescription>{t("settings.resetTokenHint")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setIsDialogOpen(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={resetTokenMutation.isPending}
              onClick={() => resetTokenMutation.mutate()}
            >
              {resetTokenMutation.isPending
                ? t("common.loading")
                : t("common.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
