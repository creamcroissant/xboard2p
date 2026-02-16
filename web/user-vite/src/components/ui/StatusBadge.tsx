import { Badge } from "@/components/ui/badge";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type StatusType = "online" | "offline" | "active" | "inactive" | "pending" | "success" | "warning" | "error";

interface StatusBadgeProps {
  status: StatusType;
  label?: string;
  showDot?: boolean;
  size?: "sm" | "md" | "lg";
  icon?: ReactNode;
}

const statusConfig: Record<
  StatusType,
  { variant: "default" | "secondary" | "destructive" | "outline" | "success" | "warning"; label: string; dotClass: string }
> = {
  online: {
    variant: "success",
    label: "Online",
    dotClass: "bg-success-foreground animate-pulse",
  },
  offline: {
    variant: "destructive",
    label: "Offline",
    dotClass: "bg-destructive-foreground",
  },
  active: {
    variant: "success",
    label: "Active",
    dotClass: "bg-success-foreground",
  },
  inactive: {
    variant: "secondary",
    label: "Inactive",
    dotClass: "bg-secondary-foreground",
  },
  pending: {
    variant: "warning",
    label: "Pending",
    dotClass: "bg-warning-foreground animate-pulse",
  },
  success: {
    variant: "success",
    label: "Success",
    dotClass: "bg-success-foreground",
  },
  warning: {
    variant: "warning",
    label: "Warning",
    dotClass: "bg-warning-foreground",
  },
  error: {
    variant: "destructive",
    label: "Error",
    dotClass: "bg-destructive-foreground",
  },
};

export default function StatusBadge({
  status,
  label,
  showDot = true,
  size = "md",
  icon,
}: StatusBadgeProps) {
  const config = statusConfig[status];
  const displayLabel = label || config.label;

  const dotSizes = {
    sm: "w-1.5 h-1.5",
    md: "w-2 h-2",
    lg: "w-2.5 h-2.5",
  };

  const badgeVariants: Record<string, string> = {
    success: "bg-green-500/15 text-green-700 dark:text-green-400 hover:bg-green-500/25 border-transparent",
    warning: "bg-yellow-500/15 text-yellow-700 dark:text-yellow-400 hover:bg-yellow-500/25 border-transparent",
    destructive: "bg-destructive/15 text-destructive hover:bg-destructive/25 border-transparent",
    secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80 border-transparent",
    default: "bg-primary text-primary-foreground hover:bg-primary/90 border-transparent",
    outline: "text-foreground border-border",
  };

  return (
    <Badge
      variant="outline"
      className={cn(
        "gap-1.5 font-medium border-0",
        badgeVariants[config.variant] || badgeVariants.default,
        size === "sm" && "px-2 py-0.5 text-xs",
        size === "md" && "px-2.5 py-0.5 text-sm",
        size === "lg" && "px-3 py-1 text-base"
      )}
    >
      {showDot ? (
        <span className={cn(dotSizes[size], "rounded-full", config.dotClass.replace("bg-success-foreground", "bg-green-500").replace("bg-destructive-foreground", "bg-red-500").replace("bg-warning-foreground", "bg-yellow-500").replace("bg-secondary-foreground", "bg-gray-500"))} />
      ) : icon ? (
        icon
      ) : null}
      {displayLabel}
    </Badge>
  );
}
