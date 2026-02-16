import type { ReactNode } from "react";

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
  size?: "sm" | "md" | "lg";
}

export default function EmptyState({
  icon,
  title,
  description,
  action,
  size = "md",
}: EmptyStateProps) {
  const sizeClasses = {
    sm: {
      container: "py-8 px-4",
      iconWrapper: "w-12 h-12 mb-3",
      iconSize: "w-6 h-6",
      title: "text-base",
      description: "text-xs",
    },
    md: {
      container: "py-16 px-6",
      iconWrapper: "w-20 h-20 mb-4",
      iconSize: "w-10 h-10",
      title: "text-lg",
      description: "text-sm",
    },
    lg: {
      container: "py-24 px-8",
      iconWrapper: "w-28 h-28 mb-6",
      iconSize: "w-14 h-14",
      title: "text-xl",
      description: "text-base",
    },
  };

  const classes = sizeClasses[size];

  return (
    <div className={`flex flex-col items-center justify-center ${classes.container} text-center`}>
      {icon && (
        <div
          className={`${classes.iconWrapper} rounded-full bg-default-100 dark:bg-default-50 flex items-center justify-center`}
        >
          <div className={`${classes.iconSize} text-default-400 dark:text-default-500`}>
            {icon}
          </div>
        </div>
      )}
      <h3 className={`${classes.title} font-semibold text-default-700 dark:text-default-300 mb-2`}>
        {title}
      </h3>
      {description && (
        <p className={`${classes.description} text-default-500 max-w-md mb-6`}>
          {description}
        </p>
      )}
      {action && <div>{action}</div>}
    </div>
  );
}
