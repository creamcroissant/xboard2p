import { Menu, LogOut, User } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate } from "react-router-dom";
import ThemeToggle from "../ThemeToggle";
import LanguageSwitcher from "../LanguageSwitcher";
import { useAuth } from "@/providers/AuthProvider";
import { ADMIN_ROUTES, ROUTES } from "@/lib/constants";
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

const ROUTE_LABEL_KEYS: Array<{ route: string; labelKey: string }> = [
  { route: ROUTES.DASHBOARD, labelKey: "nav.dashboard" },
  { route: ROUTES.SERVERS, labelKey: "nav.servers" },
  { route: ROUTES.PLANS, labelKey: "nav.plans" },
  { route: ROUTES.TRAFFIC, labelKey: "nav.traffic" },
  { route: ROUTES.KNOWLEDGE, labelKey: "nav.knowledge" },
  { route: ROUTES.SETTINGS, labelKey: "nav.settings" },
  { route: ADMIN_ROUTES.AGENTS, labelKey: "admin.nav.agents" },
  { route: ADMIN_ROUTES.USERS, labelKey: "admin.nav.users" },
  { route: ADMIN_ROUTES.PLANS, labelKey: "admin.nav.plans" },
  { route: ADMIN_ROUTES.NOTICES, labelKey: "admin.nav.notices" },
  { route: ADMIN_ROUTES.FORWARDING, labelKey: "admin.nav.forwarding" },
  { route: ADMIN_ROUTES.SYSTEM, labelKey: "admin.nav.system" },
];

function getRouteLabelKey(pathname: string) {
  const exact = ROUTE_LABEL_KEYS.find((item) => item.route === pathname);
  if (exact) {
    return exact.labelKey;
  }

  const prefix = ROUTE_LABEL_KEYS.filter((item) => pathname.startsWith(item.route))
    .sort((a, b) => b.route.length - a.route.length)
    .at(0);

  return prefix?.labelKey ?? "";
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
    <header className="sticky top-0 z-30 h-12 bg-card/80 backdrop-blur-md border-b border-border">
      <div className="flex items-center justify-between h-full px-3">
        {/* Left side */}
        <div className="flex items-center gap-3 min-w-0">
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
              <span className="text-sm font-medium text-foreground truncate block max-w-[160px] sm:max-w-[260px] md:max-w-[360px] lg:max-w-[520px]">
                {t(currentLabelKey)}
              </span>
            </div>
          )}
        </div>

        {/* Right side */}
        <div className="flex items-center gap-2">
          <LanguageSwitcher />
          <ThemeToggle />

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="gap-2 px-2">
                <span
                  className={cn(
                    "inline-flex h-8 w-8 items-center justify-center rounded-full",
                    "bg-primary text-primary-foreground text-xs font-semibold"
                  )}
                >
                  {getInitial(user?.email)}
                </span>
                <span className="hidden md:inline text-sm max-w-[120px] truncate">
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
              <DropdownMenuItem
                className="text-red-600 focus:text-red-600"
                onSelect={handleLogout}
              >
                <LogOut className="mr-2 h-4 w-4" />
                {t("nav.logout")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  );
}
