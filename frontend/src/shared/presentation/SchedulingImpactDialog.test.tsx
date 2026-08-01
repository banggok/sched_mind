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

describe("SchedulingImpactDialog", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("US-2.2 AC-31 keeps a Public Holiday save recoverable by portalling impact confirmation above the form", async () => {
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
              openProjects: [{ id: "project-1", name: "Project 1" }],
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
    expect(within(confirmation).getByText("Project 1")).toBeTruthy();

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
});
