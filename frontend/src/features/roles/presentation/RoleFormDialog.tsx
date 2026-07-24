import { useCallback, type FormEvent } from 'react'

import type { Role } from '../domain/role'
import { useDialogFocus } from './useDialogFocus'

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
  mode: 'create' | 'edit'
  role?: Role
  name: string
  error: string
  submitting: boolean
  onNameChange(name: string): void
  onSubmit(): void
  onClose(): void
}) {
  const initialName = role?.name ?? ''
  const requestClose = useCallback(() => {
    if (submitting) {
      return
    }
    if (
      name !== initialName &&
      !window.confirm('Discard your unsaved role changes?')
    ) {
      return
    }
    onClose()
  }, [initialName, name, onClose, submitting])
  const dialogRef = useDialogFocus(requestClose)

  function submit(event: FormEvent) {
    event.preventDefault()
    onSubmit()
  }

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-[#0C162D]/45 p-5 backdrop-blur-sm"
      role="presentation"
      onMouseDown={(event) => {
        if (event.currentTarget === event.target) {
          requestClose()
        }
      }}
    >
      <section
        ref={dialogRef}
        className="w-full max-w-md rounded-3xl bg-white p-7 shadow-2xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="role-form-title"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs font-extrabold tracking-[0.14em] text-[#2A93D6] uppercase">
              Role management
            </p>
            <h2
              id="role-form-title"
              className="mt-2 text-2xl font-black tracking-[-0.03em]"
            >
              {mode === 'create' ? 'Add new role' : `Edit ${role?.name}`}
            </h2>
          </div>
          <button
            type="button"
            className="grid size-9 place-items-center rounded-xl text-xl text-[#6D6E70] hover:bg-[#F7F9FC]"
            aria-label="Close role form"
            onClick={requestClose}
          >
            ×
          </button>
        </div>

        <form className="mt-7" noValidate onSubmit={submit}>
          <div className="flex gap-1 text-sm font-bold">
            <label htmlFor="role-name">Role name</label>
            <span aria-hidden="true">*</span>
          </div>
          <input
            id="role-name"
            data-autofocus
            required
            value={name}
            maxLength={101}
            aria-invalid={error ? 'true' : 'false'}
            aria-describedby={error ? 'role-name-error' : undefined}
            className="mt-2 w-full rounded-xl border border-[#D3DCE5] px-4 py-3 outline-none transition focus:border-[#2A93D6] focus:ring-4 focus:ring-[#E5F5FF]"
            placeholder="e.g. Backend Engineer"
            onChange={(event) => onNameChange(event.target.value)}
          />
          {error ? (
            <p
              id="role-name-error"
              className="mt-2 text-sm font-semibold text-rose-600"
              role="alert"
            >
              {error}
            </p>
          ) : (
            <p className="mt-2 text-sm text-[#98999A]">
              Up to 100 characters. Names must be unique.
            </p>
          )}

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
              type="submit"
              disabled={submitting}
              className="rounded-xl bg-[#2A93D6] px-5 py-3 text-sm font-bold text-white shadow-lg shadow-[#2A93D6]/20 hover:bg-[#0C4DA2] disabled:cursor-not-allowed disabled:opacity-60"
            >
              {submitting
                ? 'Saving…'
                : mode === 'create'
                  ? 'Add role'
                  : 'Save changes'}
            </button>
          </div>
        </form>
      </section>
    </div>
  )
}
