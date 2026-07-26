import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ProjectsGateway } from "../application/projectsGateway";
import { ProjectOperationError } from "../application/projectsGateway";
import type { Project } from "../domain/project";
import { ProjectsPage } from "./ProjectsPage";

const alpha: Project = {
  id: "p1",
  name: "Alpha",
  status: "open",
  autoCalculateDate: true,
  autoDependencyByAssignee: true,
  priority: 1,
  createdAt: new Date("2026-07-26"),
  updatedAt: new Date("2026-07-26"),
};

function gateway(items: Project[] = [alpha]): ProjectsGateway {
  return {
    list: vi
      .fn()
      .mockResolvedValue({ items, page: 1, pageSize: 5, total: items.length }),
    get: vi.fn(),
    create: vi.fn().mockResolvedValue(alpha),
    update: vi.fn().mockResolvedValue(alpha),
    changeStatus: vi.fn().mockResolvedValue({ ...alpha, status: "locked" }),
    movePriority: vi.fn().mockResolvedValue(alpha),
    delete: vi.fn().mockResolvedValue(undefined),
  };
}

describe("Projects page", () => {
  it("shows loading then the authoritative list and lifecycle actions", async () => {
    render(<ProjectsPage gateway={gateway()} />);
    expect(screen.getByLabelText("Loading projects")).toBeTruthy();
    expect(await screen.findByText("Alpha")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Lock" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Reopen" })).toBeNull();
  });

  it("creates a project and prevents fields outside the approved contract", async () => {
    const api = gateway([]);
    const user = userEvent.setup();
    render(<ProjectsPage gateway={api} />);
    await screen.findByText("No projects yet");
    await user.click(screen.getByRole("button", { name: "Add Project" }));
    expect(screen.queryByLabelText(/status/i)).toBeNull();
    await user.type(screen.getByLabelText("Project name"), "  Alpha  ");
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(api.create).toHaveBeenCalledWith("Alpha"));
  });

  it("asks for explicit delete confirmation", async () => {
    const api = gateway();
    const user = userEvent.setup();
    render(<ProjectsPage gateway={api} />);
    await screen.findByText("Alpha");
    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByRole("alertdialog").textContent).toContain(
      "Delete Alpha?",
    );
    await user.click(screen.getByRole("button", { name: "Delete project" }));
    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("p1"));
  });

  it("does not render operation errors inside the list", async () => {
    const api = gateway();
    api.movePriority = vi.fn().mockRejectedValue(new Error("failed"));
    const user = userEvent.setup();
    render(<ProjectsPage gateway={api} />);
    await screen.findByText("Alpha");
    await user.click(screen.getByRole("button", { name: "Move Alpha down" }));
    expect(await screen.findByRole("status")).toBeTruthy();
    expect(
      screen
        .getByRole("heading", { level: 2, name: "Projects" })
        .closest("section")?.textContent,
    ).not.toContain("The project action could not be completed. Try again.");
  });

  it("keeps a zero-task lock error in the confirmation dialog", async () => {
    const api = gateway();
    api.changeStatus = vi
      .fn()
      .mockRejectedValue(
        new ProjectOperationError(
          "PROJECT_CANNOT_LOCK_WITHOUT_TASKS",
          "project cannot lock without tasks",
          "status",
        ),
      );
    const user = userEvent.setup();
    render(<ProjectsPage gateway={api} />);
    await screen.findByText("Alpha");
    await user.click(screen.getByRole("button", { name: "Lock" }));
    await user.click(screen.getByRole("button", { name: "Lock project" }));
    expect(
      await screen.findByText(
        "Add at least one task before locking this project.",
      ),
    ).toBeTruthy();
    expect(screen.getByRole("alertdialog")).toBeTruthy();
  });
});
