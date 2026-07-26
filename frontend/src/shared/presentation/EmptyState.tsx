import type { ReactNode } from "react";

export function EmptyState({
  title,
  description,
  icon,
  action,
}: {
  title: string;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="state-panel">
      {icon ? (
        <span
          aria-hidden="true"
          className="mx-auto grid size-14 place-items-center rounded-panel bg-brand-soft text-brand"
        >
          {icon}
        </span>
      ) : null}
      <h3 className={`${icon ? "mt-5" : ""} text-dialog-title font-black`}>
        {title}
      </h3>
      {description ? (
        <p className="mx-auto mt-2 max-w-sm text-muted">{description}</p>
      ) : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  );
}
