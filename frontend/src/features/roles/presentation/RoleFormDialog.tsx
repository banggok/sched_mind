import { useCallback, type FormEvent } from "react";

import type { Role } from "../domain/role";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { FormField } from "../../../shared/presentation/FormField";

export function RoleFormDialog({
  mode,
  role,
  name,
  error,
  submitting,
  onNameChange,
  onSubmit,
  onClose,
}: {
  mode: "create" | "edit";
  role?: Role;
  name: string;
  error: string;
  submitting: boolean;
  onNameChange(name: string): void;
  onSubmit(): void;
  onClose(): void;
}) {
  const initialName = role?.name ?? "";
  const requestClose = useCallback(() => {
    if (submitting) {
      return;
    }
    if (
      name !== initialName &&
      !window.confirm("Discard your unsaved role changes?")
    ) {
      return;
    }
    onClose();
  }, [initialName, name, onClose, submitting]);
  function submit(event: FormEvent) {
    event.preventDefault();
    onSubmit();
  }

  return (
    <Dialog titleID="role-form-title" onClose={requestClose}>
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs font-extrabold tracking-widest text-brand uppercase">
            Role management
          </p>
          <h2
            id="role-form-title"
            className="mt-2 text-dialog-title font-black tracking-tight"
          >
            {mode === "create" ? "Add new role" : `Edit ${role?.name}`}
          </h2>
        </div>
        <Button
          type="button"
          variant="quiet"
          compact
          className="size-9 p-0 text-xl text-muted"
          aria-label="Close role form"
          onClick={requestClose}
        >
          ×
        </Button>
      </div>

      <form className="mt-7" noValidate onSubmit={submit}>
        <FormField
          id="role-name"
          label="Role name"
          data-autofocus
          required
          value={name}
          maxLength={101}
          error={error}
          help="Up to 100 characters. Names must be unique."
          placeholder="e.g. Backend Engineer"
          onChange={(event) => onNameChange(event.target.value)}
        />

        <div className="form-actions">
          <Button type="button" disabled={submitting} onClick={requestClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={submitting}>
            {submitting
              ? "Saving…"
              : mode === "create"
                ? "Add role"
                : "Save changes"}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
