import type { InputHTMLAttributes } from "react";

export function FormField({
  label,
  error,
  help,
  required,
  className = "",
  ...input
}: Omit<InputHTMLAttributes<HTMLInputElement>, "className"> & {
  label: string;
  error?: string;
  help?: string;
  className?: string;
}) {
  const id = input.id ?? `field-${input.name}`;
  const descriptionID = error ? `${id}-error` : help ? `${id}-help` : undefined;
  return (
    <div className={className}>
      <label
        className={`block text-label font-bold ${required ? "required-label" : ""}`.trim()}
        htmlFor={id}
      >
        {label}
      </label>
      <input
        {...input}
        id={id}
        required={required}
        aria-invalid={error ? "true" : undefined}
        aria-describedby={descriptionID}
        className="ui-input mt-2"
      />
      {error ? (
        <span
          id={`${id}-error`}
          className="mt-2 block text-label font-semibold text-danger"
          role="alert"
        >
          {error}
        </span>
      ) : help ? (
        <span
          id={`${id}-help`}
          className="mt-2 block text-label font-normal text-subtle"
        >
          {help}
        </span>
      ) : null}
    </div>
  );
}
