import { NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { X, ChevronLeft } from "lucide-react";
import { ADMIN_NAV_ITEMS, USER_NAV_ITEMS, type NavigationItemMeta } from "@/lib/constants";
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

function NavItem({
  item,
  label,
  isCollapsed,
  onClick,
}: {
  item: NavigationItemMeta;
  label: string;
  isCollapsed: boolean;
  onClick: () => void;
}) {
  const Icon = item.icon;
  const link = (
    <NavLink
      to={item.to}
      onClick={onClick}
      aria-label={isCollapsed ? label : undefined}
      className={({ isActive }) =>
        cn(
          "flex h-10 items-center rounded-md text-sm font-medium transition-colors",
          isCollapsed ? "mx-auto w-10 justify-center px-0" : "w-full justify-start px-3 py-2.5",
          isActive
            ? "bg-muted text-foreground"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        )
      }
    >
      <Icon size={20} className="h-5 w-5 shrink-0" />
      {!isCollapsed && (
        <span className="ml-3 max-w-[180px] overflow-hidden whitespace-nowrap opacity-100 transition-all duration-300 ease-in-out">
          {label}
        </span>
      )}
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
      {isOpen && <div className="fixed inset-0 z-40 bg-black/45 lg:hidden" onClick={onClose} />}

      <aside
        className={cn(
          "fixed left-0 top-0 z-50 h-dvh overflow-x-hidden border-r bg-card",
          "w-[var(--sidebar-width-expanded)] transform transition-[width,transform] duration-300 ease-in-out",
          isOpen ? "translate-x-0" : "-translate-x-full",
          "lg:relative lg:z-auto lg:flex lg:h-full lg:max-h-full lg:translate-x-0 lg:shrink-0 lg:flex-col lg:overflow-hidden",
          "lg:w-[var(--sidebar-width)]"
        )}
      >
        <div className="flex h-full min-h-0 flex-col">
          <div
            className={cn(
              "flex h-[var(--header-height)] shrink-0 items-center border-b border-border/70 transition-all duration-300",
              isCollapsed ? "justify-center px-0" : "justify-between px-4"
            )}
          >
            <Tooltip>
              <TooltipTrigger asChild>
                {isCollapsed ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-10 w-10 rounded-md p-0 hover:bg-muted/70"
                    onClick={onToggleCollapsed}
                    aria-label={t("common.expand")}
                  >
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary">
                      <span className="text-sm font-bold text-primary-foreground">X</span>
                    </div>
                  </Button>
                ) : (
                  <div className="flex items-center rounded-md px-2 py-2 transition-colors duration-200">
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary">
                      <span className="text-sm font-bold text-primary-foreground">X</span>
                    </div>
                    <span className="ml-3 max-w-[120px] overflow-hidden whitespace-nowrap text-lg font-semibold tracking-tight opacity-100 transition-all duration-300 ease-in-out">
                      XBoard
                    </span>
                  </div>
                )}
              </TooltipTrigger>
              {isCollapsed && <TooltipContent side="right">{t("common.expand")}</TooltipContent>}
            </Tooltip>

            <div
              className={cn(
                "flex items-center gap-2 overflow-hidden transition-all duration-300",
                isCollapsed ? "max-w-0 opacity-0" : "max-w-[80px] opacity-100"
              )}
            >
              <Button
                variant="ghost"
                size="icon"
                className="shrink-0 lg:hidden"
                onClick={onClose}
                aria-label={t("common.close")}
              >
                <X size={18} />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="hidden shrink-0 lg:inline-flex"
                onClick={onToggleCollapsed}
                aria-label={t("common.collapse")}
              >
                <ChevronLeft size={18} className="transition-transform duration-300" />
              </Button>
            </div>
          </div>

          <nav
            className={cn(
              "flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden",
              isCollapsed ? "items-center px-0 py-4" : "p-4"
            )}
          >
            <ul className={cn("space-y-1", isCollapsed && "flex w-full flex-col items-center")}>
              {USER_NAV_ITEMS.filter((item) => item.sidebar).map((item) => (
                <li key={item.to} className={cn(isCollapsed && "flex w-full justify-center")}>
                  <NavItem item={item} label={t(item.labelKey)} isCollapsed={isCollapsed} onClick={onClose} />
                </li>
              ))}
            </ul>

            {!isAdmin && <div className="min-h-4 flex-1" />}

            {isAdmin && (
              <div className={cn("mt-auto", isCollapsed ? "w-full pt-3" : "pt-5")}>
                <div className={cn("border-t border-border/70", isCollapsed ? "mx-auto mb-3 w-10" : "mb-5")} />
                {!isCollapsed && (
                  <p className="mb-2 max-h-6 px-3 text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground opacity-100 transition-all duration-300 ease-in-out">
                    {t("admin.nav.title")}
                  </p>
                )}
                <ul className={cn("space-y-1", isCollapsed && "flex w-full flex-col items-center")}>
                  {ADMIN_NAV_ITEMS.filter((item) => item.sidebar).map((item) => (
                    <li key={item.to} className={cn(isCollapsed && "flex w-full justify-center")}>
                      <NavItem item={item} label={t(item.labelKey)} isCollapsed={isCollapsed} onClick={onClose} />
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </nav>

          <div
            className={cn("shrink-0 border-t border-border/70 p-4 text-center", isCollapsed && "flex justify-center px-0")}
          >
            <p
              className={cn(
                "overflow-hidden whitespace-nowrap text-xs text-muted-foreground transition-all duration-300 ease-in-out",
                isCollapsed ? "flex h-6 w-10 items-center justify-center" : "max-w-full"
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
