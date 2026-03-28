import { Menu, LogOut, User } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate } from "react-router-dom";
import ThemeToggle from "../ThemeToggle";
import LanguageSwitcher from "../LanguageSwitcher";
import { useAuth } from "@/providers/AuthProvider";
import { getRouteLabelKey, ROUTES } from "@/lib/constants";
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui";
import { cn } from "@/lib/utils";

interface HeaderProps {
  onMenuClick: () => void;
}

function getInitial(email?: string) {
  if (!email) {
    return "?";
  }
  return email.trim().charAt(0).toUpperCase() || "?";
}

export default function Header({ onMenuClick }: HeaderProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuth();
  const currentLabelKey = getRouteLabelKey(location.pathname);

  const handleLogout = async () => {
    await logout();
    navigate(ROUTES.LOGIN);
  };

  return (
    <header className="sticky top-0 z-30 h-12 border-b border-border bg-card/80 backdrop-blur-md">
      <div className="flex h-full items-center justify-between px-3">
        <div className="flex min-w-0 items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            className="lg:hidden"
            onClick={onMenuClick}
            aria-label={t("nav.dashboard")}
          >
            <Menu size={18} />
          </Button>
          {currentLabelKey && (
            <div className="min-w-0">
              <span className="block max-w-[160px] truncate text-sm font-medium text-foreground sm:max-w-[260px] md:max-w-[360px] lg:max-w-[520px]">
                {t(currentLabelKey)}
              </span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-2">
          <LanguageSwitcher />
          <ThemeToggle />

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="gap-2 px-2">
                <span
                  className={cn(
                    "inline-flex h-8 w-8 items-center justify-center rounded-full",
                    "bg-primary text-xs font-semibold text-primary-foreground"
                  )}
                >
                  {getInitial(user?.email)}
                </span>
                <span className="hidden max-w-[120px] truncate text-sm md:inline">
                  {user?.email ?? "-"}
                </span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => navigate(ROUTES.SETTINGS)}>
                <User className="mr-2 h-4 w-4" />
                {t("nav.settings")}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="gap-2 text-destructive focus:text-destructive" onSelect={handleLogout}>
                <LogOut className="h-4 w-4" />
                {t("nav.logout")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  );
}
