import { NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  LayoutDashboard,
  Server,
  CreditCard,
  BarChart3,
  BookOpen,
  Settings,
  X,
  ChevronLeft,
  // Admin icons
  MonitorDot,
  Users,
  Package,
  Bell,
  Cog,
  Shuffle,
  ListChecks,
} from "lucide-react";
import { ROUTES, ADMIN_ROUTES } from "@/lib/constants";
import { useAuth } from "@/providers/AuthProvider";
import {
  Button,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui";
import { cn } from "@/lib/utils";

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
  isCollapsed: boolean;
  onToggleCollapsed: () => void;
}

const navigationItems = [
  { to: ROUTES.DASHBOARD, labelKey: "nav.dashboard", icon: LayoutDashboard },
  { to: ROUTES.SERVERS, labelKey: "nav.servers", icon: Server },
  { to: ROUTES.PLANS, labelKey: "nav.plans", icon: CreditCard },
  { to: ROUTES.TRAFFIC, labelKey: "nav.traffic", icon: BarChart3 },
  { to: ROUTES.KNOWLEDGE, labelKey: "nav.knowledge", icon: BookOpen },
  { to: ROUTES.SETTINGS, labelKey: "nav.settings", icon: Settings },
];

const adminNavigationItems = [
  { to: ADMIN_ROUTES.AGENTS, labelKey: "admin.nav.agents", icon: MonitorDot },
  { to: ADMIN_ROUTES.USERS, labelKey: "admin.nav.users", icon: Users },
  { to: ADMIN_ROUTES.PLANS, labelKey: "admin.nav.plans", icon: Package },
  { to: ADMIN_ROUTES.NOTICES, labelKey: "admin.nav.notices", icon: Bell },
  { to: ADMIN_ROUTES.KNOWLEDGE, labelKey: "admin.nav.knowledge", icon: BookOpen },
  { to: ADMIN_ROUTES.ACCESS_LOGS, labelKey: "admin.nav.accessLogs", icon: ListChecks },
  { to: ADMIN_ROUTES.FORWARDING, labelKey: "admin.nav.forwarding", icon: Shuffle },
  { to: ADMIN_ROUTES.SYSTEM, labelKey: "admin.nav.system", icon: Cog },
];

function NavItem({
  to,
  label,
  icon: Icon,
  isCollapsed,
  onClick,
}: {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  isCollapsed: boolean;
  onClick: () => void;
}) {
  const link = (
    <NavLink
      to={to}
      onClick={onClick}
      className={({ isActive }) =>
        cn(
          "flex items-center justify-start rounded-lg text-sm font-medium transition-all duration-300",
          "px-3 py-2.5",
          isActive
            ? "bg-primary text-primary-foreground"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        )
      }
    >
      <Icon size={20} className="shrink-0" />
      <span
        className={cn(
          "whitespace-nowrap overflow-hidden transition-all duration-300 ease-in-out",
          isCollapsed
            ? "max-w-0 opacity-0 ml-0"
            : "max-w-[180px] opacity-100 ml-3"
        )}
      >
        {label}
      </span>
    </NavLink>
  );

  if (!isCollapsed) {
    return link;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{link}</TooltipTrigger>
      <TooltipContent side="right">{label}</TooltipContent>
    </Tooltip>
  );
}

function AdminNavItem({
  to,
  label,
  icon: Icon,
  isCollapsed,
  onClick,
}: {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  isCollapsed: boolean;
  onClick: () => void;
}) {
  const link = (
    <NavLink
      to={to}
      onClick={onClick}
      className={({ isActive }) =>
        cn(
          "flex items-center justify-start rounded-lg text-sm font-medium transition-all duration-300",
          "px-3 py-2.5",
          isActive
            ? "bg-amber-500/20 text-amber-600 dark:text-amber-400"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        )
      }
    >
      <Icon size={20} className="shrink-0" />
      <span
        className={cn(
          "whitespace-nowrap overflow-hidden transition-all duration-300 ease-in-out",
          isCollapsed
            ? "max-w-0 opacity-0 ml-0"
            : "max-w-[180px] opacity-100 ml-3"
        )}
      >
        {label}
      </span>
    </NavLink>
  );

  if (!isCollapsed) {
    return link;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{link}</TooltipTrigger>
      <TooltipContent side="right">{label}</TooltipContent>
    </Tooltip>
  );
}

export default function Sidebar({
  isOpen,
  onClose,
  isCollapsed,
  onToggleCollapsed,
}: SidebarProps) {
  const { t } = useTranslation();
  const { isAdmin } = useAuth();

  return (
    <TooltipProvider>
      {/* Mobile overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden"
          onClick={onClose}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed top-0 left-0 z-50 h-full bg-card border-r border-border",
          "transform transition-[width,transform] duration-300 ease-in-out",
          "w-[var(--sidebar-width-expanded)]",
          isOpen ? "translate-x-0" : "-translate-x-full",
          "lg:relative lg:translate-x-0 lg:z-auto lg:flex lg:flex-col lg:shrink-0",
          "lg:w-[var(--sidebar-width)]"
        )}
      >
        <div className="flex flex-col h-full">
          {/* Header */}
          <div
            className={cn(
              "flex items-center h-16 border-b border-border transition-all duration-300",
              isCollapsed ? "justify-start px-4" : "justify-between px-4"
            )}
          >
            {/* Logo - clickable to expand when collapsed */}
            <Tooltip>
              <TooltipTrigger asChild>
                <div
                  className={cn(
                    "flex items-center px-3 py-2.5 rounded-lg transition-all duration-300",
                    isCollapsed && "cursor-pointer hover:bg-muted"
                  )}
                  onClick={isCollapsed ? onToggleCollapsed : undefined}
                  role={isCollapsed ? "button" : undefined}
                  aria-label={isCollapsed ? t("common.expand") : undefined}
                >
                  <div className="w-5 h-5 rounded bg-gradient-to-br from-primary to-secondary flex items-center justify-center shrink-0">
                    <span className="text-white font-bold text-xs">X</span>
                  </div>
                  <span
                    className={cn(
                      "font-bold text-lg whitespace-nowrap overflow-hidden transition-all duration-300 ease-in-out",
                      isCollapsed
                        ? "max-w-0 opacity-0 ml-0"
                        : "max-w-[120px] opacity-100 ml-3"
                    )}
                  >
                    XBoard
                  </span>
                </div>
              </TooltipTrigger>
              {isCollapsed && (
                <TooltipContent side="right">{t("common.expand")}</TooltipContent>
              )}
            </Tooltip>

            {/* Collapse button - only visible when expanded */}
            <div
              className={cn(
                "flex items-center gap-2 transition-all duration-300 overflow-hidden",
                isCollapsed ? "max-w-0 opacity-0" : "max-w-[80px] opacity-100"
              )}
            >
              <Button
                variant="ghost"
                size="icon"
                className="lg:hidden shrink-0"
                onClick={onClose}
                aria-label={t("common.close")}
              >
                <X size={18} />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="hidden lg:inline-flex shrink-0"
                onClick={onToggleCollapsed}
                aria-label={t("common.collapse")}
              >
                <ChevronLeft size={18} className="transition-transform duration-300" />
              </Button>
            </div>
          </div>

          {/* Navigation */}
          <nav className="flex-1 flex flex-col overflow-y-auto p-4">
            {/* User Navigation */}
            <ul className="space-y-1">
              {navigationItems.map((item) => (
                <li key={item.to}>
                  <NavItem
                    to={item.to}
                    label={t(item.labelKey)}
                    icon={item.icon}
                    isCollapsed={isCollapsed}
                    onClick={onClose}
                  />
                </li>
              ))}
            </ul>

            {/* Spacer for non-admin users to push footer down */}
            {!isAdmin && <div className="flex-1 min-h-4" />}

            {/* Admin Navigation */}
            {isAdmin && (
              <div className="mt-auto">
                <div className="my-4 border-t border-border" />
                <p
                  className={cn(
                    "px-3 mb-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider transition-all duration-300 ease-in-out",
                    isCollapsed
                      ? "max-h-0 opacity-0 mb-0 overflow-hidden"
                      : "max-h-6 opacity-100"
                  )}
                >
                  {t("admin.nav.title")}
                </p>
                <ul className="space-y-1">
                  {adminNavigationItems.map((item) => (
                    <li key={item.to}>
                      <AdminNavItem
                        to={item.to}
                        label={t(item.labelKey)}
                        icon={item.icon}
                        isCollapsed={isCollapsed}
                        onClick={onClose}
                      />
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </nav>

          {/* Footer */}
          <div className="border-t border-border text-center p-4">
            <p
              className={cn(
                "text-xs text-muted-foreground transition-all duration-300 ease-in-out overflow-hidden whitespace-nowrap",
                isCollapsed ? "max-w-[20px]" : "max-w-full"
              )}
            >
              {isCollapsed ? "©" : `© ${new Date().getFullYear()} XBoard`}
            </p>
          </div>
        </div>
      </aside>
    </TooltipProvider>
  );
}
