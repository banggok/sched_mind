import { useCallback } from 'react'

import type { Role } from '../domain/role'
import { useDialogFocus } from './useDialogFocus'

export function DeleteRoleDialog({
  role,
  error,
  submitting,
  onCancel,
  onConfirm,
}: {
  role: Role
  error: string
  submitting: boolean
  onCancel(): void
  onConfirm(): void
}) {
  const requestClose = useCallback(() => {
    if (!submitting) {
      onCancel()
    }
  }, [onCancel, submitting])
  const dialogRef = useDialogFocus(requestClose)

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-[#0C162D]/45 p-5 backdrop-blur-sm">
      <section
        ref={dialogRef}
        className="w-full max-w-md rounded-3xl bg-white p-7 shadow-2xl"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="delete-role-title"
      >
        <span className="grid size-12 place-items-center rounded-2xl bg-rose-50 text-xl text-rose-600">
          !
        </span>
        <h2
          id="delete-role-title"
          className="mt-5 text-2xl font-black tracking-[-0.03em]"
        >
          Delete {role.name}?
        </h2>
        <p className="mt-3 leading-7 text-[#6D6E70]">
          This role will be permanently removed. Roles assigned to members
          cannot be deleted.
        </p>
        {error ? (
          <p className="mt-4 text-sm font-semibold text-rose-600" role="alert">
            {error}
          </p>
        ) : null}
        <div className="mt-7 flex justify-end gap-3">
          <button
            type="button"
            disabled={submitting}
            className="rounded-xl border border-[#D3DCE5] px-5 py-3 text-sm font-bold hover:bg-[#F7F9FC] disabled:cursor-not-allowed disabled:opacity-60"
            onClick={requestClose}
          >
            Cancel
          </button>
          <button
            type="button"
            data-autofocus
            disabled={submitting}
            className="rounded-xl bg-rose-600 px-5 py-3 text-sm font-bold text-white hover:bg-rose-700 disabled:opacity-60"
            onClick={onConfirm}
          >
            {submitting ? 'Deleting…' : 'Delete role'}
          </button>
        </div>
      </section>
    </div>
  )
}
