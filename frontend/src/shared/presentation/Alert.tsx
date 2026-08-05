import type { ReactNode } from "react";

export function Alert({
  tone,
  children,
  className = "",
}: {
  tone: "success" | "danger" | "warning";
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`rounded-control p-4 font-bold ${tone === "success" ? "bg-success-soft text-success" : tone === "warning" ? "bg-warning-soft text-warning" : "bg-danger-soft text-danger"} ${className}`.trim()}
      role={tone === "danger" ? "alert" : "status"}
    >
      {children}
    </div>
  );
}
