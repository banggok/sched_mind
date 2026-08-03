import type { ReactNode } from "react";

export function PageContent({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={`page-container ${className}`.trim()}>{children}</div>;
}
