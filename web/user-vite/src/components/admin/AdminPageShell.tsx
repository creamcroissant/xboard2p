import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface AdminPageShellProps {
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
  toolbar?: ReactNode;
  stats?: ReactNode;
  children: ReactNode;
}

export default function AdminPageShell({
  title,
  description,
  actions,
  toolbar,
  stats,
  children,
}: AdminPageShellProps) {
  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold text-foreground">{title}</h1>
          {description ? <div className="text-sm text-muted-foreground">{description}</div> : null}
        </div>
        {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
      </div>

      {toolbar ? <div className={cn("flex flex-col gap-3")}>{toolbar}</div> : null}
      {stats}
      {children}
    </div>
  );
}
