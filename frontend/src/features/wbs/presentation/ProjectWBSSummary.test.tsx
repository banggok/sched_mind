import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { ProjectWBSSummary } from "./ProjectWBSSummary";

function task(id: string, effortMinutes: number, actualEnd?: string): WBSNode {
  return {
    id,
    projectId: "project",
    name: id,
    position: 1,
    hasChildren: false,
    executable: {
      effortMinutes,
      lagDays: 0,
      executionTimeline: { start: "2026-08-01", end: "2026-08-02" },
      commitmentTimeline: { start: "2026-08-01", end: "2026-08-03" },
      actualStart: actualEnd,
      actualEnd,
    },
    children: [],
  };
}

function reader(
  tree: WBSGateway["tree"],
  subscribeToConfirmedChanges?: WBSGateway["subscribeToConfirmedChanges"],
): Pick<WBSGateway, "tree" | "subscribeToConfirmedChanges"> {
  return { tree, subscribeToConfirmedChanges };
}

describe("US-4.3 Project WBS summary loader", () => {
  it("AC-31 opens with a local loading state and renders the confirmed tree", async () => {
    const pending = deferred<WBSNode[]>();
    const gateway = reader(vi.fn().mockReturnValue(pending.promise));
    render(<ProjectWBSSummary projectId="project" gateway={gateway} />);

    expect(screen.getByLabelText("Loading project summary")).toBeTruthy();
    await act(async () => pending.resolve([task("done", 480, "2026-08-02")]));

    expect(
      await screen.findByText("8 of 8 hours completed (100%)"),
    ).toBeTruthy();
    expect(gateway.tree).toHaveBeenCalledWith(
      "project",
      expect.any(AbortSignal),
    );
  });

  it("AC-30 renders the ordinary empty Project state without misleading zero aggregates", async () => {
    const gateway = reader(vi.fn().mockResolvedValue([]));
    render(<ProjectWBSSummary projectId="project" gateway={gateway} />);

    expect(
      await screen.findByText("No tasks are available for this project."),
    ).toBeTruthy();
    expect(screen.queryByText(/0 of 0/)).toBeNull();
    expect(screen.queryByText(/%/)).toBeNull();
    expect(screen.queryByText("Not scheduled")).toBeNull();
  });

  it("AC-32 exposes local Retry and recovers without a mutation request", async () => {
    const tree = vi
      .fn<WBSGateway["tree"]>()
      .mockRejectedValueOnce(new Error("network detail"))
      .mockResolvedValueOnce([task("todo", 480)]);
    const gateway = reader(tree);
    const user = userEvent.setup();
    render(<ProjectWBSSummary projectId="project" gateway={gateway} />);

    expect(
      await screen.findByText(
        "Project summary could not be loaded. Try again.",
      ),
    ).toBeTruthy();
    expect(screen.queryByText("network detail")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Retry" }));

    expect(await screen.findByText("0 of 8 hours completed (0%)")).toBeTruthy();
    expect(tree).toHaveBeenCalledTimes(2);
  });

  it("AC-20 AC-21 AC-33 refreshes on a confirmed projection change and ignores the older response", async () => {
    const older = deferred<WBSNode[]>();
    const newer = deferred<WBSNode[]>();
    const tree = vi
      .fn<WBSGateway["tree"]>()
      .mockReturnValueOnce(older.promise)
      .mockReturnValueOnce(newer.promise);
    let notifyConfirmedChange: () => void = () => undefined;
    const gateway = reader(tree, (listener) => {
      notifyConfirmedChange = listener;
      return () => {
        notifyConfirmedChange = () => undefined;
      };
    });
    render(<ProjectWBSSummary projectId="project" gateway={gateway} />);
    await waitFor(() => expect(tree).toHaveBeenCalledTimes(1));

    act(() => notifyConfirmedChange());
    await waitFor(() => expect(tree).toHaveBeenCalledTimes(2));

    await act(async () => newer.resolve([task("new", 960, "2026-08-02")]));
    expect(
      await screen.findByText("16 of 16 hours completed (100%)"),
    ).toBeTruthy();

    await act(async () => older.resolve([task("old", 480)]));
    expect(screen.getByText("16 of 16 hours completed (100%)")).toBeTruthy();
    expect(screen.queryByText("0 of 8 hours completed (0%)")).toBeNull();
  });
});

function deferred<T>(): {
  promise: Promise<T>;
  resolve(value: T): void;
} {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((accept) => {
    resolve = accept;
  });
  return { promise, resolve };
}
