import type { ReactNode } from "react";

export function PageContent({ children }: { children: ReactNode }) {
  return <div className="page-container">{children}</div>;
}
