import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { CapacityOverridesGateway } from "../application/capacityOverridesGateway";
import type { CapacityOverride } from "../domain/capacityOverride";
import { CapacityOverridesPanel } from "./CapacityOverridesPanel";
const member = {
  id: "m",
  name: "Harry",
  role: { id: "r", name: "Backend" },
  dailyCapacity: 8,
  bufferPercentage: 20,
  commitmentCapacity: 6.5,
  createdAt: new Date(),
  updatedAt: new Date(),
};
function gateway(items: CapacityOverride[] = []): CapacityOverridesGateway {
  return {
    list: vi
      .fn()
      .mockResolvedValue({ items, page: 1, pageSize: 5, total: items.length }),
    get: vi
      .fn()
      .mockImplementation(async (_member, id) =>
        items.find((item) => item.id === id),
      ),
    create: vi.fn().mockResolvedValue(items[0]),
    update: vi.fn().mockResolvedValue(items[0]),
    delete: vi.fn().mockResolvedValue(undefined),
  };
}
describe("CapacityOverridesPanel", () => {
  it("selects a date range with two calendar clicks and resets an earlier second date", async () => {
    const user = userEvent.setup();
    const api = gateway();
    render(
      <CapacityOverridesPanel
        member={member}
        gateway={api}
        onClose={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("status", { name: "Loading capacity overrides" }),
    ).toBeTruthy();
    expect(await screen.findByText("No capacity overrides yet")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "+ Add Override" }));
    await user.type(screen.getByLabelText("Description"), "Training");
    expect(screen.queryByText("No capacity overrides yet")).toBeNull();
    expect(screen.queryByText("Effective Date")).toBeNull();
    expect(screen.queryByRole("button", { name: "+ Add Override" })).toBeNull();
    await user.click(
      screen.getByRole("button", {
        name: "Date range: Select start and end date",
      }),
    );
    await user.click(screen.getByRole("button", { name: "2026-07-04" }));
    expect(screen.getByText(/Select an end date/)).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "2026-07-03" }));
    expect(
      screen.getByRole("button", { name: /Jul 3, 2026 — Select end date/ }),
    ).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "2026-07-03" }));
    expect(
      screen.queryByRole("dialog", { name: "Choose date range" }),
    ).toBeNull();
    await user.type(screen.getByLabelText("Capacity (hours)"), "4");
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(api.create).toHaveBeenCalledWith("m", {
        description: "Training",
        startDate: "2026-07-03",
        endDate: "2026-07-03",
        capacity: 4,
      }),
    );
  });
  it("renders scoped values and confirms deletion", async () => {
    const user = userEvent.setup();
    const item = {
      id: "o",
      teamMemberId: "m",
      description: "Training",
      startDate: "2026-07-03",
      endDate: "2026-07-04",
      capacity: 4,
      createdAt: new Date(),
      updatedAt: new Date(),
    };
    const api = gateway([item]);
    render(
      <CapacityOverridesPanel
        member={member}
        gateway={api}
        onClose={vi.fn()}
      />,
    );
    expect(await screen.findByText("Training")).toBeTruthy();
    expect(screen.getByText(/4 hours\/day/)).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(api.delete).not.toHaveBeenCalled();
    await user.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: "Cancel",
      }),
    );
    expect(api.delete).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    await user.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: "Delete",
      }),
    );
    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("m", "o"));
  });

  it("makes edit mode exclusive and restores the list after cancel", async () => {
    const user = userEvent.setup();
    const item: CapacityOverride = {
      id: "o",
      teamMemberId: "m",
      description: "Training",
      startDate: "2026-07-03",
      endDate: "2026-07-04",
      capacity: 4,
      createdAt: new Date(),
      updatedAt: new Date(),
    };
    render(
      <CapacityOverridesPanel
        member={member}
        gateway={gateway([item])}
        onClose={vi.fn()}
      />,
    );

    expect(await screen.findByText("Training")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Edit" }));

    expect(await screen.findByText("Edit override")).toBeTruthy();
    expect(screen.queryByText(/4 hours\/day/)).toBeNull();
    expect(screen.queryByText("Effective Date")).toBeNull();
    expect(screen.queryByRole("button", { name: "Edit" })).toBeNull();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByText(/4 hours\/day/)).toBeTruthy();
    expect(screen.getByText("Effective Date")).toBeTruthy();
  });
});

describe("Date range calendar placement", () => {
  it("opens above when there is not enough viewport space below", async () => {
    const user = userEvent.setup();
    const rectangle = vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockImplementation(function (this: HTMLElement) {
        if (this.getAttribute("aria-label") === "Choose date range")
          return {
            top: 0,
            bottom: 320,
            left: 0,
            right: 352,
            width: 352,
            height: 320,
            x: 0,
            y: 0,
            toJSON() {},
          };
        if (this.getAttribute("aria-haspopup") === "dialog")
          return {
            top: 700,
            bottom: 750,
            left: 0,
            right: 352,
            width: 352,
            height: 50,
            x: 0,
            y: 700,
            toJSON() {},
          };
        return {
          top: 0,
          bottom: 0,
          left: 0,
          right: 0,
          width: 0,
          height: 0,
          x: 0,
          y: 0,
          toJSON() {},
        };
      });
    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: 800,
    });
    render(
      <CapacityOverridesPanel
        member={member}
        gateway={gateway()}
        onClose={vi.fn()}
      />,
    );
    await screen.findByText("No capacity overrides yet");
    await user.click(screen.getByRole("button", { name: "+ Add Override" }));
    await user.type(screen.getByLabelText("Description"), "Training");
    await user.click(
      screen.getByRole("button", {
        name: "Date range: Select start and end date",
      }),
    );
    await waitFor(() =>
      expect(
        screen
          .getByRole("dialog", { name: "Choose date range" })
          .getAttribute("data-placement"),
      ).toBe("above"),
    );
    rectangle.mockRestore();
  });
});

describe("Capacity input normalization", () => {
  it("accepts only digits and one decimal point, preserves the draft, and rounds on blur and submit", async () => {
    const user = userEvent.setup();
    const api = gateway();
    render(
      <CapacityOverridesPanel
        member={member}
        gateway={api}
        onClose={vi.fn()}
      />,
    );
    await screen.findByText("No capacity overrides yet");
    await user.click(screen.getByRole("button", { name: "+ Add Override" }));
    await user.type(screen.getByLabelText("Description"), "Training");
    await user.click(
      screen.getByRole("button", {
        name: "Date range: Select start and end date",
      }),
    );
    await user.click(screen.getByRole("button", { name: "2026-07-03" }));
    await user.click(screen.getByRole("button", { name: "2026-07-03" }));
    const capacity = screen.getByLabelText(
      "Capacity (hours)",
    ) as HTMLInputElement;
    await user.type(capacity, "7a,.3.");
    expect(capacity.value).toBe("7.3");
    await user.tab();
    expect(capacity.value).toBe("7.5");
    await user.clear(capacity);
    await user.type(capacity, "7.8");
    expect(capacity.value).toBe("7.8");
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(api.create).toHaveBeenCalledWith("m", {
        description: "Training",
        startDate: "2026-07-03",
        endDate: "2026-07-03",
        capacity: 8,
      }),
    );
  });
});

describe("Effective Date filtering", () => {
  it("requests page one, shows no-results, and clears back to the normal list", async () => {
    const user = userEvent.setup();
    const item = {
      id: "o",
      teamMemberId: "m",
      description: "Support",
      startDate: "2026-07-26",
      endDate: "2026-07-28",
      capacity: 3,
      createdAt: new Date(),
      updatedAt: new Date(),
    };
    const api = gateway([item]);
    (api.list as ReturnType<typeof vi.fn>).mockImplementation(
      async (_member, query) => ({
        items: query.effectiveDate ? [] : [item],
        page: query.page,
        pageSize: 5,
        total: query.effectiveDate ? 0 : 1,
      }),
    );
    render(
      <CapacityOverridesPanel
        member={member}
        gateway={api}
        onClose={vi.fn()}
      />,
    );
    expect(await screen.findByText("Support")).toBeTruthy();
    await user.click(
      screen.getByRole("button", { name: "Effective Date: Select date" }),
    );
    await user.click(screen.getByRole("button", { name: "2026-07-27" }));
    expect(
      await screen.findByText("No capacity override applies on Jul 27, 2026."),
    ).toBeTruthy();
    expect(api.list).toHaveBeenLastCalledWith(
      "m",
      { page: 1, pageSize: 5, effectiveDate: "2026-07-27" },
      undefined,
    );
    await user.click(
      screen.getByRole("button", { name: "Clear Effective Date" }),
    );
    expect(await screen.findByText("Support")).toBeTruthy();
    expect(api.list).toHaveBeenLastCalledWith(
      "m",
      { page: 1, pageSize: 5, effectiveDate: undefined },
      undefined,
    );
  });
  it("preserves Effective Date during pagination", async () => {
    const user = userEvent.setup();
    const items = Array.from({ length: 5 }, (_, index) => ({
      id: `o-${index}`,
      teamMemberId: "m",
      description: `Training ${index}`,
      startDate: "2026-07-26",
      endDate: "2026-07-28",
      capacity: 3,
      createdAt: new Date(),
      updatedAt: new Date(),
    }));
    const api = gateway(items);
    (api.list as ReturnType<typeof vi.fn>).mockImplementation(
      async (_member, query) => ({
        items,
        page: query.page,
        pageSize: 5,
        total: 10,
      }),
    );
    render(
      <CapacityOverridesPanel
        member={member}
        gateway={api}
        onClose={vi.fn()}
      />,
    );
    await screen.findByText("Page 1 of 2");
    await user.click(
      screen.getByRole("button", { name: "Effective Date: Select date" }),
    );
    await user.click(screen.getByRole("button", { name: "2026-07-27" }));
    await waitFor(() =>
      expect(api.list).toHaveBeenLastCalledWith(
        "m",
        { page: 1, pageSize: 5, effectiveDate: "2026-07-27" },
        undefined,
      ),
    );
    await user.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() =>
      expect(api.list).toHaveBeenLastCalledWith(
        "m",
        { page: 2, pageSize: 5, effectiveDate: "2026-07-27" },
        undefined,
      ),
    );
  });
});
