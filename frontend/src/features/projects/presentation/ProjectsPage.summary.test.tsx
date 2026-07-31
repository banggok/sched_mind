import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { describe, expect, it, vi } from "vitest";
import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import type { WBSGateway } from "../../wbs/application/wbsGateway";
import type { WBSNode } from "../../wbs/domain/wbs";
import { ProjectWBSSummary } from "../../wbs/presentation/ProjectWBSSummary";
import type { ProjectsGateway } from "../application/projectsGateway";
import type { Project } from "../domain/project";
import { ProjectsPage } from "./ProjectsPage";

const alpha: Project = {
  id: "project",
  name: "Alpha",
  status: "open",
  autoCalculateDate: true,
  automaticScheduling: true,
  schedulingStartDate: "2026-07-20",
  projectBuffer: 20,
  startDate: "2020-01-01",
  endDate: "2030-01-01",
  scheduleVersion: 0,
  priority: 1,
  createdAt: new Date("2026-07-01T00:00:00Z"),
  updatedAt: new Date("2026-07-01T00:00:00Z"),
};

function projectGateway(projects: Project[] = [alpha]): ProjectsGateway {
  return {
    list: vi.fn().mockResolvedValue({
      items: projects,
      page: 1,
      pageSize: 5,
      total: projects.length,
    }),
    get: vi.fn(),
    create: vi.fn().mockResolvedValue(alpha),
    update: vi.fn().mockResolvedValue(alpha),
    changeStatus: vi.fn().mockResolvedValue(alpha),
    movePriority: vi.fn().mockResolvedValue(alpha),
    updateSettings: vi.fn().mockResolvedValue(alpha),
    delete: vi.fn().mockResolvedValue(undefined),
  };
}

function task(id: string, input: Partial<WBSNode["executable"]> = {}): WBSNode {
  return {
    id,
    projectId: "project",
    name: id,
    position: 1,
    hasChildren: false,
    executable: {
      lagDays: 0,
      executionTimeline: {},
      commitmentTimeline: {},
      ...input,
    },
    children: [],
  };
}

function group(id: string, children: WBSNode[]): WBSNode {
  return {
    id,
    projectId: "project",
    name: id,
    position: 1,
    hasChildren: true,
    executable: {
      effortMinutes: 99_999,
      actualEnd: "2020-01-01",
      lagDays: 0,
      executionTimeline: { start: "2020-01-01", end: "2030-01-01" },
      commitmentTimeline: { start: "2020-01-01", end: "2030-01-01" },
    },
    children,
  };
}

function multiRootTree(): WBSNode[] {
  return [
    group("delivery", [
      task("analysis", {
        effortMinutes: 960,
        actualEnd: "2026-08-03",
        executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
        commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
      }),
      group("nested", [
        task("build", {
          effortMinutes: 1440,
          executionTimeline: { start: "2026-08-04", end: "2026-08-08" },
          commitmentTimeline: { start: "2026-08-04", end: "2026-08-10" },
        }),
      ]),
    ]),
    group("release", [
      task("verify", {
        effortMinutes: 480,
        actualEnd: "2026-08-12",
        commitmentTimeline: { start: "2026-08-11", end: "2026-08-12" },
      }),
    ]),
    task("handover", { actualEnd: "2026-08-13" }),
  ];
}

function summaryGateway(
  tree: WBSGateway["tree"],
  subscribeToConfirmedChanges?: WBSGateway["subscribeToConfirmedChanges"],
): Pick<WBSGateway, "tree" | "subscribeToConfirmedChanges"> {
  return { tree, subscribeToConfirmedChanges };
}

function renderPage(
  projects: Project[],
  gateway: Pick<WBSGateway, "tree" | "subscribeToConfirmedChanges">,
) {
  const projectsGateway = projectGateway(projects);
  const view = render(
    <ProjectsPage
      gateway={projectsGateway}
      renderProjectSummary={(project) => (
        <ProjectWBSSummary
          key={project.id}
          projectId={project.id}
          gateway={gateway}
        />
      )}
    />,
  );
  return { projectsGateway, view };
}

async function openEdit(projectName = "Alpha"): Promise<HTMLElement> {
  const user = userEvent.setup();
  await screen.findByText(projectName);
  await user.click(screen.getByRole("button", { name: "Edit" }));
  return screen.findByRole("dialog", { name: `Edit ${projectName}` });
}

describe("US-4.3 Project Summary acceptance workflow", () => {
  it("AC-28 AC-29 AC-34 AC-35 AC-38 renders the exact recursive multi-root summary in the wide Edit form", async () => {
    const tree = vi.fn().mockResolvedValue(multiRootTree());
    const gateway = summaryGateway(tree);
    const { projectsGateway } = renderPage([alpha], gateway);

    const dialog = await openEdit();
    expect(dialog.className).toContain("dialog-panel-wide");
    expect(within(dialog).getByLabelText("Project name")).toBe(
      document.activeElement,
    );

    const summaryRegion = within(dialog)
      .getByRole("heading", { name: "Project Summary" })
      .closest<HTMLElement>("section");
    if (!summaryRegion) throw new Error("Project Summary region not found");
    expect(
      await within(summaryRegion).findByText("24 of 48 hours completed (50%)"),
    ).toBeTruthy();
    expect(
      within(summaryRegion)
        .getAllByRole("heading", { level: 4 })
        .map((heading) => heading.textContent),
    ).toEqual([
      "Execution Timeline",
      "Commitment Timeline",
      "Effort Completion",
    ]);
    expect(
      await within(summaryRegion).findByText(
        `${formatDateOnly("2026-08-01")} – ${formatDateOnly("2026-08-08")}`,
      ),
    ).toBeTruthy();
    expect(
      within(summaryRegion).getByText(
        `${formatDateOnly("2026-08-01")} – ${formatDateOnly("2026-08-12")}`,
      ),
    ).toBeTruthy();
    expect(
      within(summaryRegion).getByText("2 of 4 tasks scheduled"),
    ).toBeTruthy();
    expect(
      within(summaryRegion).getByText("3 of 4 tasks scheduled"),
    ).toBeTruthy();
    expect(
      within(summaryRegion).getByText(
        "1 task without effort is excluded from this calculation.",
      ),
    ).toBeTruthy();
    expect(within(summaryRegion).queryByText(/2020/)).toBeNull();
    expect(within(summaryRegion).queryByText(/2030/)).toBeNull();
    expect(tree).toHaveBeenCalledWith("project", expect.any(AbortSignal));
    expect(projectsGateway.create).not.toHaveBeenCalled();
    expect(projectsGateway.update).not.toHaveBeenCalled();

    const summaryGrid = within(summaryRegion).getByLabelText(
      "Project summary details",
    );
    expect(summaryGrid.className).toContain("min-w-0");
    expect(summaryGrid.classList.contains("grid-cols-3")).toBe(false);
    expect(summaryGrid.classList.contains("lg:grid-cols-3")).toBe(true);
    expect(
      within(summaryRegion).getByText("24 of 48 hours completed (50%)")
        .className,
    ).toContain("break-words");
    const formActions = within(dialog).getByRole("button", {
      name: "Cancel",
    }).parentElement;
    if (!formActions) throw new Error("Project form actions not found");
    expect(
      summaryRegion.compareDocumentPosition(formActions) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();

    const results = await axe.run(dialog, {
      rules: { "color-contrast": { enabled: false } },
    });
    expect(results.violations).toEqual([]);
  });

  it("AC-30 AC-36 keeps Add standard and summary-free while Edit shows the ordinary empty state", async () => {
    const user = userEvent.setup();
    const gateway = summaryGateway(vi.fn().mockResolvedValue([]));
    renderPage([alpha], gateway);
    await screen.findByText("Alpha");

    await user.click(screen.getByRole("button", { name: "+ Add Project" }));
    let dialog = await screen.findByRole("dialog", { name: "Add project" });
    expect(dialog.className).not.toContain("dialog-panel-wide");
    expect(within(dialog).queryByText("Project Summary")).toBeNull();
    expect(gateway.tree).not.toHaveBeenCalled();
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    dialog = await openEdit();
    const summaryRegion = within(dialog)
      .getByRole("heading", { name: "Project Summary" })
      .closest<HTMLElement>("section");
    if (!summaryRegion) throw new Error("Project Summary region not found");
    expect(
      await within(summaryRegion).findByText(
        "No tasks are available for this project.",
      ),
    ).toBeTruthy();
    expect(within(summaryRegion).queryByText(/0 of 0/)).toBeNull();
    expect(within(summaryRegion).queryByText(/%/)).toBeNull();
    expect(within(summaryRegion).queryByText("Not scheduled")).toBeNull();
  });

  it("AC-31 AC-32 keeps loading, error, and Retry local without clearing draft or blocking form actions", async () => {
    const first = deferred<WBSNode[]>();
    const second = deferred<WBSNode[]>();
    const tree = vi
      .fn<WBSGateway["tree"]>()
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    const gateway = summaryGateway(tree);
    const { projectsGateway } = renderPage([alpha], gateway);
    const user = userEvent.setup();

    const dialog = await openEdit();
    const name = within(dialog).getByLabelText("Project name");
    expect(name).toBe(document.activeElement);
    expect(
      within(dialog).getByLabelText("Loading project summary"),
    ).toBeTruthy();
    await user.clear(name);
    await user.type(name, "Alpha revised");
    expect((name as HTMLInputElement).value).toBe("Alpha revised");

    await act(async () => first.reject(new Error("private network detail")));
    expect(
      await within(dialog).findByText(
        "Project summary could not be loaded. Try again.",
      ),
    ).toBeTruthy();
    expect(within(dialog).queryByText("private network detail")).toBeNull();
    expect((name as HTMLInputElement).value).toBe("Alpha revised");
    expect(within(dialog).getByRole("button", { name: "Save" })).toBeTruthy();
    expect(within(dialog).getByRole("button", { name: "Cancel" })).toBeTruthy();

    await user.click(within(dialog).getByRole("button", { name: "Retry" }));
    await act(async () => second.resolve([]));
    expect(
      await within(dialog).findByText(
        "No tasks are available for this project.",
      ),
    ).toBeTruthy();
    expect((name as HTMLInputElement).value).toBe("Alpha revised");

    await user.click(within(dialog).getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(projectsGateway.update).toHaveBeenCalledWith(
        "project",
        "Alpha revised",
        true,
        "2026-07-20",
        20,
      ),
    );
  });

  it("AC-17 AC-18 AC-20 AC-21 AC-33 refreshes completion and prevents an older Project response from restoring stale values", async () => {
    const older = deferred<WBSNode[]>();
    const newer = deferred<WBSNode[]>();
    const reopened = deferred<WBSNode[]>();
    const tree = vi
      .fn<WBSGateway["tree"]>()
      .mockReturnValueOnce(older.promise)
      .mockReturnValueOnce(newer.promise)
      .mockReturnValueOnce(reopened.promise);
    let notifyConfirmedChange: () => void = () => undefined;
    const gateway = summaryGateway(tree, (listener) => {
      notifyConfirmedChange = listener;
      return () => {
        notifyConfirmedChange = () => undefined;
      };
    });
    renderPage([alpha], gateway);

    const dialog = await openEdit();
    const name = within(dialog).getByLabelText("Project name");
    fireEvent.change(name, { target: { value: "Draft remains" } });
    await waitFor(() => expect(tree).toHaveBeenCalledTimes(1));

    act(() => notifyConfirmedChange());
    await waitFor(() => expect(tree).toHaveBeenCalledTimes(2));
    await act(async () =>
      newer.resolve([
        task("new", { effortMinutes: 960, actualEnd: "2026-08-01" }),
      ]),
    );
    expect(
      await within(dialog).findByText("16 of 16 hours completed (100%)"),
    ).toBeTruthy();

    await act(async () => older.resolve([task("old", { effortMinutes: 480 })]));
    expect(
      within(dialog).getByText("16 of 16 hours completed (100%)"),
    ).toBeTruthy();
    expect(
      within(dialog).queryByText("0 of 8 hours completed (0%)"),
    ).toBeNull();
    expect((name as HTMLInputElement).value).toBe("Draft remains");

    act(() => notifyConfirmedChange());
    await waitFor(() => expect(tree).toHaveBeenCalledTimes(3));
    await act(async () =>
      reopened.resolve([task("new", { effortMinutes: 960 })]),
    );
    expect(
      await within(dialog).findByText("0 of 16 hours completed (0%)"),
    ).toBeTruthy();
    expect((name as HTMLInputElement).value).toBe("Draft remains");
  });

  it("AC-37 preserves Open, Locked, and Closed field permissions while summary remains read-only", async () => {
    for (const status of ["open", "locked", "closed"] as const) {
      const current = { ...alpha, status, name: `Alpha ${status}` };
      const gateway = summaryGateway(
        vi.fn().mockResolvedValue([task("known", { effortMinutes: 480 })]),
      );
      const { view } = renderPage([current], gateway);
      const dialog = await openEdit(current.name);
      expect(
        await within(dialog).findByText("0 of 8 hours completed (0%)"),
      ).toBeTruthy();
      expect(
        within(dialog).getByLabelText("Project name").hasAttribute("disabled"),
      ).toBe(status === "closed");
      expect(
        within(dialog)
          .getByRole("switch", { name: "Automatic scheduling" })
          .hasAttribute("disabled"),
      ).toBe(status !== "open");
      expect(
        within(dialog).queryByRole("button", { name: "Save" }) !== null,
      ).toBe(status !== "closed");
      view.unmount();
    }
  });
});

function deferred<T>(): {
  promise: Promise<T>;
  resolve(value: T): void;
  reject(reason: unknown): void;
} {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((accept, deny) => {
    resolve = accept;
    reject = deny;
  });
  return { promise, resolve, reject };
}
