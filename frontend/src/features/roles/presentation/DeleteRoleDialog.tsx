import { useCallback } from "react";

import type { Role } from "../domain/role";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";

export function DeleteRoleDialog({
  role,
  error,
  submitting,
  onCancel,
  onConfirm,
}: {
  role: Role;
  error: string;
  submitting: boolean;
  onCancel(): void;
  onConfirm(): void;
}) {
  const requestClose = useCallback(() => {
    if (!submitting) {
      onCancel();
    }
  }, [onCancel, submitting]);
  return (
    <Dialog
      titleID="delete-role-title"
      onClose={requestClose}
      kind="alertdialog"
      closeOnBackdrop={false}
    >
      <span className="grid size-12 place-items-center rounded-panel bg-danger-soft text-xl text-danger">
        !
      </span>
      <h2
        id="delete-role-title"
        className="mt-5 text-dialog-title font-black tracking-tight"
      >
        Delete {role.name}?
      </h2>
      <p className="mt-3 leading-7 text-muted">
        This role will be permanently removed. Active team members or tasks
        using this role prevent deletion.
      </p>
      {error ? (
        <p className="mt-4 text-sm font-semibold text-danger" role="alert">
          {error}
        </p>
      ) : null}
      <div className="form-actions">
        <Button type="button" disabled={submitting} onClick={requestClose}>
          Cancel
        </Button>
        <Button
          type="button"
          data-autofocus
          variant="danger-solid"
          loading={submitting}
          onClick={onConfirm}
        >
          {submitting ? "Deleting…" : "Delete role"}
        </Button>
      </div>
    </Dialog>
  );
}
