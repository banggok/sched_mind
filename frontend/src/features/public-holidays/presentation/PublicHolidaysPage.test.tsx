import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { PublicHolidaysGateway } from "../application/publicHolidaysGateway";
import type { PublicHoliday } from "../domain/publicHoliday";
import { PublicHolidaysPage } from "./PublicHolidaysPage";
const value: PublicHoliday = {
  id: "h",
  startDate: "2026-08-17",
  endDate: "2026-08-17",
  description: "Independence Day",
  createdAt: new Date(),
  updatedAt: new Date(),
};
function gateway(items: PublicHoliday[] = []): PublicHolidaysGateway {
  return {
    list: vi
      .fn()
      .mockResolvedValue({ items, page: 1, pageSize: 5, total: items.length }),
    get: vi.fn().mockImplementation(async () => value),
    create: vi.fn().mockResolvedValue(value),
    update: vi.fn().mockResolvedValue(value),
    delete: vi.fn().mockResolvedValue(undefined),
    calendar: vi.fn().mockResolvedValue([]),
  };
}

async function navigateCalendarToDate(
  user: ReturnType<typeof userEvent.setup>,
  targetDate: string,
) {
  const target = new Date(`${targetDate}T00:00:00Z`);
  const current = new Date();
  const monthDifference =
    (target.getUTCFullYear() - current.getUTCFullYear()) * 12 +
    target.getUTCMonth() -
    current.getUTCMonth();
  const navigationLabel = monthDifference < 0 ? "Previous month" : "Next month";
  for (let index = 0; index < Math.abs(monthDifference); index += 1)
    await user.click(screen.getByRole("button", { name: navigationLabel }));
}
describe("PublicHolidaysPage", () => {
  it("renders loading and empty states then validates and creates", async () => {
    const user = userEvent.setup();
    const api = gateway();
    render(<PublicHolidaysPage gateway={api} />);
    expect(
      screen.getByRole("status", { name: "Loading public holidays" }),
    ).toBeTruthy();
    expect(await screen.findByText("No public holidays yet")).toBeTruthy();
    await user.click(
      screen.getByRole("button", { name: "+ Add Public Holiday" }),
    );
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText("Start date is required")).toBeTruthy();
    expect(screen.getByRole("dialog")).toBeTruthy();
    await user.click(
      screen.getByRole("button", {
        name: "Date range: Select start and end date",
      }),
    );
    await navigateCalendarToDate(user, "2026-08-17");
    await user.click(screen.getByRole("button", { name: "2026-08-17" }));
    await user.click(screen.getByRole("button", { name: "2026-08-17" }));
    expect(screen.queryByText("Start date is required")).toBeNull();
    await user.type(screen.getByLabelText("Description"), " Independence Day ");
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(api.create).toHaveBeenCalledWith({
        startDate: "2026-08-17",
        endDate: "2026-08-17",
        description: "Independence Day",
      }),
    );
    expect(
      screen.getByRole("button", { name: "Dismiss notification" }),
    ).toBeTruthy();
    await user.click(
      screen.getByRole("button", { name: "Dismiss notification" }),
    );
    expect(screen.queryByText("Public holiday added.")).toBeNull();
  });
  it("shows data, filters exact date, clears, and confirms delete", async () => {
    const user = userEvent.setup();
    const api = gateway([value]);
    render(<PublicHolidaysPage gateway={api} />);
    expect(await screen.findByText("Independence Day")).toBeTruthy();
    await user.click(
      screen.getByRole("button", { name: "Holiday Date: Select date" }),
    );
    await navigateCalendarToDate(user, "2026-08-17");
    await user.click(screen.getByRole("button", { name: "2026-08-17" }));
    await waitFor(() =>
      expect(api.list).toHaveBeenLastCalledWith(
        { page: 1, pageSize: 5, holidayDate: "2026-08-17" },
        undefined,
      ),
    );
    await user.click(screen.getByRole("button", { name: "Clear" }));
    await user.click(
      screen.getByRole("button", { name: "Delete Independence Day" }),
    );
    expect(api.delete).not.toHaveBeenCalled();
    const dialog = screen.getByRole("alertdialog");
    expect(within(dialog).getByText(/Aug 17, 2026/)).toBeTruthy();
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(api.delete).not.toHaveBeenCalled();
    await user.click(
      screen.getByRole("button", { name: "Delete Independence Day" }),
    );
    await user.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: "Delete",
      }),
    );
    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("h"));
  });
  it("shows a distinct filtered no-results state", async () => {
    const user = userEvent.setup();
    const api = gateway();
    render(<PublicHolidaysPage gateway={api} />);
    await screen.findByText("No public holidays yet");
    await user.click(
      screen.getByRole("button", { name: "Holiday Date: Select date" }),
    );
    await navigateCalendarToDate(user, "2026-08-17");
    await user.click(screen.getByRole("button", { name: "2026-08-17" }));
    expect(
      await screen.findByText(
        "No public holiday is configured for Aug 17, 2026.",
      ),
    ).toBeTruthy();
  });

  it("recovers from a list failure with Retry", async () => {
    const user = userEvent.setup();
    const api = gateway([value]);
    (api.list as ReturnType<typeof vi.fn>)
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({
        items: [value],
        page: 1,
        pageSize: 5,
        total: 1,
      });
    render(<PublicHolidaysPage gateway={api} />);
    expect(
      await screen.findByText(
        "Public holidays could not be loaded. Please try again.",
      ),
    ).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByText("Independence Day")).toBeTruthy();
  });
});
