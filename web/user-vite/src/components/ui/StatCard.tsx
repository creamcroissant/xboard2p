import type { ReactNode } from "react";
import { TrendingUp, TrendingDown, Minus } from "lucide-react";
import { Card, CardContent } from "@/components/ui";
import { cn } from "@/lib/utils";

interface StatCardProps {
  title: string;
  value: string | number;
  hint?: string;
  icon?: ReactNode;
  children?: ReactNode;
  className?: string;
  trend?: {
    value: number;
    label?: string;
  };
  variant?: "default" | "primary" | "success" | "warning" | "danger";
}

export default function StatCard({
  title,
  value,
  hint,
  icon,
  children,
  className = "",
  trend,
  variant = "default",
}: StatCardProps) {
  const variantStyles = {
    default: {
      iconText: "text-muted-foreground",
    },
    primary: {
      iconText: "text-primary",
    },
    success: {
      iconText: "text-emerald-600",
    },
    warning: {
      iconText: "text-amber-600",
    },
    danger: {
      iconText: "text-red-600",
    },
  };

  const styles = variantStyles[variant];

  const getTrendIcon = () => {
    if (!trend) return null;
    if (trend.value > 0) {
      return <TrendingUp className="h-3 w-3" />;
    }
    if (trend.value < 0) {
      return <TrendingDown className="h-3 w-3" />;
    }
    return <Minus className="h-3 w-3" />;
  };

  const getTrendColor = () => {
    if (!trend) return "";
    if (trend.value > 0) return "text-emerald-600";
    if (trend.value < 0) return "text-red-600";
    return "text-muted-foreground";
  };

  return (
    <Card className={cn("transition-shadow", className)}>
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          {icon && (
            <div
              className={cn(
                "flex h-9 w-9 items-center justify-center flex-shrink-0",
                styles.iconText
              )}
            >
              {icon}
            </div>
          )}
          <div className="flex-1 min-w-0">
            <p className="text-sm text-muted-foreground">{title}</p>
            <div className="mt-1 flex items-baseline gap-2">
              <p className="text-xl font-semibold text-foreground truncate">
                {value}
              </p>
              {trend && (
                <div className={cn("flex items-center gap-1 text-xs", getTrendColor())}>
                  {getTrendIcon()}
                  <span>{Math.abs(trend.value)}%</span>
                  {trend.label && (
                    <span className="text-muted-foreground">{trend.label}</span>
                  )}
                </div>
              )}
            </div>
            {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
            {children}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
