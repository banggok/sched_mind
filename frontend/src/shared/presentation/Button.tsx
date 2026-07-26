import type { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonVariant =
  "primary" | "secondary" | "quiet" | "danger" | "danger-solid";

export function Button({
  variant = "secondary",
  compact = false,
  loading = false,
  className = "",
  children,
  disabled,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  compact?: boolean;
  loading?: boolean;
  children: ReactNode;
}) {
  return (
    <button
      {...props}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={`ui-button ui-button-${variant} ${compact ? "ui-button-compact" : ""} ${className}`.trim()}
    >
      {children}
    </button>
  );
}
