import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { schedulingImpactFetch } from "../infrastructure/schedulingImpactFetch";
import { Dialog } from "./Dialog";
import { SchedulingImpactDialog } from "./SchedulingImpactDialog";

function PublicHolidaySaveFixture() {
  const [status, setStatus] = useState("idle");
  return (
    <>
      <Dialog titleID="holiday-form-title" onClose={() => undefined}>
        <h2 id="holiday-form-title">Add public holiday</h2>
        <button
          type="button"
          onClick={() => {
            setStatus("saving");
            void schedulingImpactFetch("/api/public-holidays", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({
                startDate: "2026-08-03",
                endDate: "2026-08-03",
                description: "test",
              }),
            }).then(() => setStatus("saved"));
          }}
        >
          Save public holiday
        </button>
      </Dialog>
      <SchedulingImpactDialog />
      <output aria-label="save status">{status}</output>
    </>
  );
}

function GroupReopenFixture() {
  const [status, setStatus] = useState("idle");
  return (
    <>
      <button
        type="button"
        onClick={() => {
          setStatus("reopening");
          void schedulingImpactFetch(
            "/api/projects/project-a/wbs/group-a/status",
            { method: "POST" },
            {
              reopenAll: () =>
                fetch("/api/projects/project-a/wbs/group-a/reopen-all", {
                  method: "POST",
                }),
            },
          ).then(() => setStatus("reopened"));
        }}
      >
        Reopen Group
      </button>
      <SchedulingImpactDialog />
      <output aria-label="group reopen status">{status}</output>
    </>
  );
}
describe("SchedulingImpactDialog", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("US-6.2 D01/D05 AC-20/AC-24A shows only backend-classified timeline impacts while keeping Public Holiday confirmation recoverable", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: "SCHEDULING_IMPACT_CONFIRMATION_REQUIRED",
            message: "scheduling impact confirmation is required",
            details: {
              token: "impact-token",
              lockedProjects: [],
              openProjects: [
                { id: "timeline-project", name: "Timeline Project" },
              ],
            },
          }),
          {
            status: 409,
            headers: { "Content-Type": "application/json" },
          },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: { id: "holiday" } }), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    render(<PublicHolidaySaveFixture />);

    await user.click(
      screen.getByRole("button", { name: "Save public holiday" }),
    );

    const confirmation = await screen.findByRole("alertdialog", {
      name: "Review scheduling impact",
    });
    const confirmationOverlay = confirmation.parentElement;
    expect(confirmationOverlay?.className).toContain("dialog-overlay-nested");
    expect(confirmationOverlay?.parentElement).toBe(document.body);
    expect(document.activeElement?.closest('[role="alertdialog"]')).toBe(
      confirmation,
    );
    expect(screen.getByLabelText("save status").textContent).toBe("saving");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(within(confirmation).getByText("Timeline Project")).toBeTruthy();
    expect(
      within(confirmation).queryByText("Allocation-only Project"),
    ).toBeNull();
    expect(
      within(confirmation).getByText(/timeline-impacted projects/i),
    ).toBeTruthy();

    await user.click(
      within(confirmation).getByRole("button", { name: "Confirm and save" }),
    );

    await waitFor(() =>
      expect(screen.getByLabelText("save status").textContent).toBe("saved"),
    );
    expect(fetchMock).toHaveBeenCalledTimes(2);
    const confirmedRequest = fetchMock.mock.calls[1]?.[1];
    expect(
      new Headers(confirmedRequest?.headers).get("X-Scheduling-Impact-Token"),
    ).toBe("impact-token");
  });

  it("US-4.4 AC-22/AC-25 shows qualified Group closure and keeps Reopen all keyboard focus", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: "SCHEDULING_SCOPE_REOPEN_CLOSURE_REQUIRED",
            message: "reopen closure required",
            details: {
              token: "group-token",
              lockedProjects: [{ id: "project-b", name: "Project B" }],
              openProjects: [],
              lockedGroups: [
                {
                  id: "group-b",
                  projectId: "project-b",
                  name: "Backend",
                  path: "Project B / Platform / Backend",
                },
              ],
              openGroups: [],
            },
          }),
          {
            status: 409,
            headers: { "Content-Type": "application/json" },
          },
        ),
      )
      .mockResolvedValueOnce(new Response(null, { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    render(<GroupReopenFixture />);
    await user.click(screen.getByRole("button", { name: "Reopen Group" }));

    const dialog = await screen.findByRole("alertdialog", {
      name: "Reopen required scheduling scopes",
    });
    expect(within(dialog).getByText("Project B")).toBeTruthy();
    expect(
      within(dialog).getByText("Project B / Platform / Backend"),
    ).toBeTruthy();
    const reopenAll = within(dialog).getByRole("button", {
      name: "Reopen all",
    });
    expect(document.activeElement).toBe(reopenAll);

    await user.click(reopenAll);
    await waitFor(() =>
      expect(screen.getByLabelText("group reopen status").textContent).toBe(
        "reopened",
      ),
    );
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
