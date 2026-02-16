import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Plus, Search, MoreVertical, Ban, Trash2 } from "lucide-react";
import { QUERY_KEYS } from "@/lib/constants";
import { getUsers, createUser, toggleUserBan, deleteUser } from "@/api/admin";
import type { AdminUser, CreateUserRequest } from "@/types";
import { formatBytes } from "@/components/admin";
import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Input,
  Loading,
  Pagination,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui";

export default function UserList() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [newUser, setNewUser] = useState<CreateUserRequest>({
    email: "",
    password: "",
  });

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: [...QUERY_KEYS.ADMIN_USERS, page, search],
    queryFn: () => getUsers({ page, page_size: 20, search: search || undefined }),
  });

  const createMutation = useMutation({
    mutationFn: createUser,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_USERS });
      setIsDialogOpen(false);
      setNewUser({ email: "", password: "" });
      toast.success(t("admin.users.createSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("admin.users.createError"), { description: err.message });
    },
  });

  const banMutation = useMutation({
    mutationFn: ({ id, banned }: { id: number; banned: boolean }) => toggleUserBan(id, banned),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_USERS });
      toast.success(t("admin.users.updateSuccess"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteUser,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.ADMIN_USERS });
      toast.success(t("admin.users.deleteSuccess"));
    },
  });

  const users: AdminUser[] = data?.data || [];
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / 20);

  const formatDate = (timestamp?: number) => {
    if (!timestamp) return "-";
    return new Date(timestamp * 1000).toLocaleDateString();
  };

  const handleDialogChange = (open: boolean) => {
    setIsDialogOpen(open);
    if (!open) {
      setNewUser({ email: "", password: "" });
    }
  };

  const handleCreate = () => {
    createMutation.mutate(newUser);
  };

  if (isLoading) return <Loading />;

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <p className="text-sm text-destructive">{t("admin.users.loadError")}</p>
        <Button variant="outline" onClick={() => refetch()}>
          {t("common.retry")}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">{t("admin.users.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("admin.users.total", { count: total })}
          </p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <div className="relative w-full sm:w-64">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="h-10 pl-9"
              placeholder={t("admin.users.searchPlaceholder")}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && refetch()}
            />
          </div>
          <Button onClick={() => setIsDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("admin.users.add")}
          </Button>
        </div>
      </div>

      <div className="overflow-x-auto">
        <Table aria-label={t("admin.users.title")}>
          <TableHeader>
            <TableRow>
              <TableHead>{t("admin.users.email")}</TableHead>
              <TableHead>{t("admin.users.plan")}</TableHead>
              <TableHead>{t("admin.users.traffic")}</TableHead>
              <TableHead>{t("admin.users.expiredAt")}</TableHead>
              <TableHead>{t("admin.users.status")}</TableHead>
              <TableHead>{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                  {t("admin.users.empty")}
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <span className="font-medium text-foreground">
                        {user.email || user.username || "-"}
                      </span>
                      {user.is_admin && <Badge variant="warning">{t("admin.users.admin")}</Badge>}
                    </div>
                  </TableCell>
                  <TableCell>{user.plan_name || "-"}</TableCell>
                  <TableCell>
                    {formatBytes(user.u + user.d)} / {formatBytes(user.transfer_enable)}
                  </TableCell>
                  <TableCell>{formatDate(user.expired_at)}</TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        user.banned
                          ? "danger"
                          : user.status === 1
                          ? "success"
                          : "default"
                      }
                    >
                      {user.banned
                        ? t("admin.users.banned")
                        : user.status === 1
                        ? t("admin.users.active")
                        : t("admin.users.inactive")}
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
                        <DropdownMenuItem
                          className="gap-2"
                          onSelect={() => banMutation.mutate({ id: user.id, banned: !user.banned })}
                        >
                          <Ban className="h-4 w-4" />
                          {user.banned ? t("admin.users.unban") : t("admin.users.ban")}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="gap-2 text-red-600 focus:text-red-600"
                          onSelect={() => deleteMutation.mutate(user.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                          {t("common.delete")}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {totalPages > 1 && (
        <div className="flex justify-center">
          <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
        </div>
      )}

      <Dialog open={isDialogOpen} onOpenChange={handleDialogChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("admin.users.addTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.users.email")}</label>
              <Input
                placeholder="user@example.com"
                value={newUser.email || ""}
                onChange={(event) => setNewUser({ ...newUser, email: event.target.value })}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.users.username")}</label>
              <Input
                placeholder={t("admin.users.usernamePlaceholder")}
                value={newUser.username || ""}
                onChange={(event) => setNewUser({ ...newUser, username: event.target.value })}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t("admin.users.password")}</label>
              <Input
                type="password"
                placeholder="••••••••"
                value={newUser.password}
                onChange={(event) => setNewUser({ ...newUser, password: event.target.value })}
                required
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => handleDialogChange(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleCreate} disabled={createMutation.isPending}>
              {createMutation.isPending ? t("common.loading") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
