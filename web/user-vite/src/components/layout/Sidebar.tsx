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
      className={({ isActive }) =>
        cn(
          "flex items-center justify-start rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-300",
          isActive
            ? "bg-primary text-primary-foreground"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        )
      }
    >
      <Icon size={20} className="shrink-0" />
      <span
        className={cn(
          "overflow-hidden whitespace-nowrap transition-all duration-300 ease-in-out",
          isCollapsed ? "ml-0 max-w-0 opacity-0" : "ml-3 max-w-[180px] opacity-100"
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
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={onClose}
        />
      )}

      <aside
        className={cn(
          "fixed left-0 top-0 z-50 h-full border-r border-border bg-card",
          "w-[var(--sidebar-width-expanded)] transform transition-[width,transform] duration-300 ease-in-out",
          isOpen ? "translate-x-0" : "-translate-x-full",
          "lg:relative lg:z-auto lg:flex lg:translate-x-0 lg:shrink-0 lg:flex-col",
          "lg:w-[var(--sidebar-width)]"
        )}
      >
        <div className="flex h-full flex-col">
          <div
            className={cn(
              "flex h-16 items-center border-b border-border px-4 transition-all duration-300",
              isCollapsed ? "justify-start" : "justify-between"
            )}
          >
            <Tooltip>
              <TooltipTrigger asChild>
                <div
                  className={cn(
                    "flex items-center rounded-lg px-3 py-2.5 transition-all duration-300",
                    isCollapsed && "cursor-pointer hover:bg-muted"
                  )}
                  onClick={isCollapsed ? onToggleCollapsed : undefined}
                  role={isCollapsed ? "button" : undefined}
                  aria-label={isCollapsed ? t("common.expand") : undefined}
                >
                  <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded bg-gradient-to-br from-primary to-secondary">
                    <span className="text-xs font-bold text-white">X</span>
                  </div>
                  <span
                    className={cn(
                      "overflow-hidden whitespace-nowrap text-lg font-bold transition-all duration-300 ease-in-out",
                      isCollapsed ? "ml-0 max-w-0 opacity-0" : "ml-3 max-w-[120px] opacity-100"
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

          <nav className="flex flex-1 flex-col overflow-y-auto p-4">
            <ul className="space-y-1">
              {USER_NAV_ITEMS.filter((item) => item.sidebar).map((item) => (
                <li key={item.to}>
                  <NavItem
                    item={item}
                    label={t(item.labelKey)}
                    isCollapsed={isCollapsed}
                    onClick={onClose}
                  />
                </li>
              ))}
            </ul>

            {!isAdmin && <div className="min-h-4 flex-1" />}

            {isAdmin && (
              <div className="mt-auto">
                <div className="my-4 border-t border-border" />
                <p
                  className={cn(
                    "mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground transition-all duration-300 ease-in-out",
                    isCollapsed ? "mb-0 max-h-0 overflow-hidden opacity-0" : "max-h-6 opacity-100"
                  )}
                >
                  {t("admin.nav.title")}
                </p>
                <ul className="space-y-1">
                  {ADMIN_NAV_ITEMS.filter((item) => item.sidebar).map((item) => (
                    <li key={item.to}>
                      <NavItem
                        item={item}
                        label={t(item.labelKey)}
                        isCollapsed={isCollapsed}
                        onClick={onClose}
                      />
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </nav>

          <div className="border-t border-border p-4 text-center">
            <p
              className={cn(
                "overflow-hidden whitespace-nowrap text-xs text-muted-foreground transition-all duration-300 ease-in-out",
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
