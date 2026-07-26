import type { ReactNode } from "react";

export function ListSurface({
  title,
  controls,
  children,
}: {
  title: string;
  controls?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="list-surface">
      <div className="list-surface-header">
        <h2 className="text-section-title font-black">{title}</h2>
        {controls}
      </div>
      {children}
    </section>
  );
}
