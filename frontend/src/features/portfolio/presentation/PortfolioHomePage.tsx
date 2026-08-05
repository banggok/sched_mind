import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type FormEvent,
} from "react";
import { createPortal } from "react-dom";
import { PageContent } from "../../../app/PageContent";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { EmptyState } from "../../../shared/presentation/EmptyState";
import { FormField } from "../../../shared/presentation/FormField";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import { Toast } from "../../../shared/presentation/Toast";
import {
  currentScheduleProjectionVersion,
  subscribeScheduleProjectionVersion,
} from "../../../shared/infrastructure/scheduleProjectionClock";
import {
  WBSOperationError,
  type WBSGateway,
} from "../../wbs/application/wbsGateway";
import type { PortfolioGateway } from "../application/portfolioGateway";
import {
  PortfolioOperationError,
  type PortfolioDependency,
  type PortfolioHoliday,
  type PortfolioProject,
  type PortfolioProjection,
  type PortfolioProjectionResult,
  type PortfolioRow,
  type SavedPortfolioFilter,
} from "../domain/portfolio";

const systemFilterID = "__all-active-projects__";
const unassignedRoleKey = "__unassigned__";
const dayWidth = 34;
const rowHeight = 40;
const ganttHeaderHeight = 84;
const maximumRenderedRows = 80;
const rowOverscan = 8;
const shortMonthNames = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

const initialColumns: GridColumn[] = [
  { key: "wbs", label: "WBS", width: 48, minimum: 44, maximum: 96 },
  { key: "name", label: "Name", width: 190, minimum: 140, maximum: 520 },
  { key: "role", label: "Role", width: 100, minimum: 80, maximum: 260 },
  {
    key: "assignee",
    label: "Assignee",
    width: 100,
    minimum: 80,
    maximum: 260,
  },
  { key: "effort", label: "Effort", width: 64, minimum: 56, maximum: 120 },
  { key: "start", label: "Start", width: 92, minimum: 84, maximum: 160 },
  { key: "end", label: "End", width: 92, minimum: 84, maximum: 160 },
];

type ValidRange = { from: string; to: string };
type BarFractions = { start: number; end: number };
type WBSOpenRequest = {
  projectId: string;
  nodeId?: string;
  createParentId?: string | null;
  moveNodeId?: string;
};
export type HomeProjectCommandKind = "lock" | "close" | "reopen" | "delete";
export type HomeRowFocusRequest = {
  key: number;
  rowId: string;
  minimumProjectionVersion?: number;
};
type RoleOption = { key: string; label: string };
type GridColumn = {
  key: "wbs" | "name" | "role" | "assignee" | "effort" | "start" | "end";
  label: string;
  width: number;
  minimum: number;
  maximum: number;
};

export function PortfolioHomePage({
  gateway,
  wbsGateway,
  onOpenProject,
  onProjectCommand,
  onOpenWBS,
  focusRequest,
  onFocusRequestHandled,
}: {
  gateway: PortfolioGateway;
  wbsGateway: WBSGateway;
  onOpenProject(projectId: string): void;
  onProjectCommand(kind: HomeProjectCommandKind, projectId: string): void;
  onOpenWBS(request: WBSOpenRequest): void;
  focusRequest?: HomeRowFocusRequest;
  onFocusRequestHandled?(key: number): void;
}) {
  const [projects, setProjects] = useState<PortfolioProject[]>([]);
  const [filters, setFilters] = useState<SavedPortfolioFilter[]>([]);
  const [activeFilterID, setActiveFilterID] = useState(systemFilterID);
  const [selectedIDs, setSelectedIDs] = useState<string[]>([]);
  const [roleFilterActive, setRoleFilterActive] = useState(false);
  const [selectedRoleKeys, setSelectedRoleKeys] = useState<string[]>([]);
  const [filterSavedFilterDraft, setFilterSavedFilterDraft] =
    useState(systemFilterID);
  const [filterProjectionDraft, setFilterProjectionDraft] =
    useState<PortfolioProjection>("execution");
  const [filterProjectDraft, setFilterProjectDraft] = useState<string[]>([]);
  const [filterRoleDraft, setFilterRoleDraft] = useState<string[]>([]);
  const [projectSearch, setProjectSearch] = useState("");
  const [projection, setProjection] =
    useState<PortfolioProjection>("execution");
  const [validRange, setValidRange] = useState<ValidRange>(currentMonthRange);
  const [queryRange] = useState<ValidRange>(portfolioQueryRange);
  const [portfolio, setPortfolio] = useState<PortfolioProjectionResult>();
  const [portfolioProjectionVersion, setPortfolioProjectionVersion] =
    useState<number>();
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const [metadataLoading, setMetadataLoading] = useState(true);
  const [projectionLoading, setProjectionLoading] = useState(true);
  const [metadataError, setMetadataError] = useState("");
  const [projectionError, setProjectionError] = useState("");
  const [reloadMetadata, setReloadMetadata] = useState(0);
  const [reloadProjection, setReloadProjection] = useState(0);
  const [dialog, setDialog] = useState<"filter" | "save-as" | "delete">();
  const [filterName, setFilterName] = useState("");
  const [filterError, setFilterError] = useState("");
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState("");
  const [actionBusyRowID, setActionBusyRowID] = useState("");
  const [pendingDelete, setPendingDelete] = useState<PortfolioRow>();
  const [localFocusRequest, setLocalFocusRequest] =
    useState<HomeRowFocusRequest>();
  const localFocusRequestVersion = useRef(0);
  const activeFilterIDRef = useRef(activeFilterID);
  const metadataInitializedRef = useRef(false);

  useEffect(() => {
    activeFilterIDRef.current = activeFilterID;
  }, [activeFilterID]);

  useEffect(
    () =>
      subscribeScheduleProjectionVersion(() => {
        setReloadMetadata((value) => value + 1);
        setReloadProjection((value) => value + 1);
      }),
    [],
  );

  useEffect(() => {
    const controller = new AbortController();
    setMetadataLoading(true);
    setMetadataError("");
    void Promise.all([
      gateway.activeProjects(controller.signal),
      gateway.savedFilters(controller.signal),
    ])
      .then(([activeProjects, savedFilters]) => {
        setProjects(activeProjects);
        setFilters(sortFilters(savedFilters));
        const active = new Set(activeProjects.map((project) => project.id));
        const currentFilterID = activeFilterIDRef.current;
        const latest = savedFilters.find(
          (filter) => filter.id === currentFilterID,
        );

        if (!metadataInitializedRef.current) {
          metadataInitializedRef.current = true;
          if (currentFilterID === systemFilterID) {
            setSelectedIDs(activeProjects.map((project) => project.id));
          } else if (latest) {
            setSelectedIDs(latest.projectIds.filter((id) => active.has(id)));
          } else {
            setActiveFilterID(systemFilterID);
            setSelectedIDs(activeProjects.map((project) => project.id));
          }
          return;
        }

        if (currentFilterID === systemFilterID) {
          setSelectedIDs(activeProjects.map((project) => project.id));
        } else if (!latest) {
          setActiveFilterID(systemFilterID);
          setSelectedIDs(activeProjects.map((project) => project.id));
        } else {
          setSelectedIDs((current) =>
            current.filter((projectID) => active.has(projectID)),
          );
        }
      })
      .catch((reason: unknown) => {
        if (!isAbort(reason))
          setMetadataError("Home filters could not be loaded. Try again.");
      })
      .finally(() => {
        if (!controller.signal.aborted) setMetadataLoading(false);
      });
    return () => controller.abort();
  }, [gateway, reloadMetadata]);

  useEffect(() => {
    if (metadataLoading || selectedIDs.length === 0) {
      if (!metadataLoading) setProjectionLoading(false);
      return;
    }
    const controller = new AbortController();
    const requestedProjectionVersion = currentScheduleProjectionVersion();
    setProjectionLoading(true);
    setProjectionError("");
    void gateway
      .projection(
        selectedIDs,
        projection,
        queryRange.from,
        queryRange.to,
        controller.signal,
      )
      .then((result) => {
        setPortfolio(result);
        setPortfolioProjectionVersion(requestedProjectionVersion);
      })
      .catch((reason: unknown) => {
        if (!isAbort(reason))
          setProjectionError("Portfolio timeline could not be loaded.");
      })
      .finally(() => {
        if (!controller.signal.aborted) setProjectionLoading(false);
      });
    return () => controller.abort();
  }, [
    gateway,
    metadataLoading,
    projection,
    queryRange.from,
    queryRange.to,
    reloadProjection,
    selectedIDs,
  ]);

  const activeFilter = filters.find((filter) => filter.id === activeFilterID);
  const draftSavedFilter = filters.find(
    (filter) => filter.id === filterSavedFilterDraft,
  );
  const portfolioTitle = activeFilter?.name ?? "All Active Projects";
  const roleOptions = useMemo(
    () => collectRoleOptions(portfolio?.rows ?? []),
    [portfolio?.rows],
  );
  const effectiveRoleKeys = useMemo(
    () =>
      roleFilterActive
        ? selectedRoleKeys
        : roleOptions.map((option) => option.key),
    [roleFilterActive, roleOptions, selectedRoleKeys],
  );
  const roleAdjustedRows = useMemo(
    () =>
      applyRoleFilter(
        portfolio?.rows ?? [],
        new Set(effectiveRoleKeys),
        roleFilterActive,
      ),
    [effectiveRoleKeys, portfolio?.rows, roleFilterActive],
  );
  const visibleRows = useMemo(
    () => visiblePortfolioRows(roleAdjustedRows, collapsed),
    [collapsed, roleAdjustedRows],
  );
  const activeFocusRequest = focusRequest ?? localFocusRequest;
  const projectionReadyFocusRequest =
    activeFocusRequest?.minimumProjectionVersion === undefined ||
    (portfolioProjectionVersion !== undefined &&
      portfolioProjectionVersion >= activeFocusRequest.minimumProjectionVersion)
      ? activeFocusRequest
      : undefined;
  const visibleFocusRequest = useMemo(
    () =>
      focusRequestForRoleAdjustedRows(
        projectionReadyFocusRequest,
        portfolio?.rows ?? [],
        roleAdjustedRows,
      ),
    [portfolio?.rows, projectionReadyFocusRequest, roleAdjustedRows],
  );

  useEffect(() => {
    if (!projectionReadyFocusRequest || !portfolio) return;
    const target = portfolio.rows.find(
      (row) => row.id === projectionReadyFocusRequest.rowId,
    );
    if (!target) return;
    const byID = new Map(portfolio.rows.map((row) => [row.id, row]));
    const ancestors = new Set<string>();
    const projectRow = portfolio.rows.find(
      (row) => row.kind === "project" && row.projectId === target.projectId,
    );
    if (projectRow) ancestors.add(projectRow.id);
    let parentID = target.parentId;
    while (parentID) {
      ancestors.add(parentID);
      parentID = byID.get(parentID)?.parentId;
    }
    setCollapsed((current) => {
      if (![...ancestors].some((id) => current.has(id))) return current;
      const next = new Set(current);
      ancestors.forEach((id) => next.delete(id));
      return next;
    });
  }, [portfolio, projectionReadyFocusRequest]);

  useEffect(() => {
    if (focusRequest) setLocalFocusRequest(undefined);
  }, [focusRequest]);
  const restrictiveRoleFilter =
    roleFilterActive && effectiveRoleKeys.length < roleOptions.length;
  const filterProjectDraftSet = useMemo(
    () => new Set(filterProjectDraft),
    [filterProjectDraft],
  );
  const filterRoleDraftSet = useMemo(
    () => new Set(filterRoleDraft),
    [filterRoleDraft],
  );
  const visibleProjects = useMemo(() => {
    const needle = projectSearch.trim().toLocaleLowerCase();
    return needle
      ? projects.filter((project) =>
          project.name.toLocaleLowerCase().includes(needle),
        )
      : projects;
  }, [projectSearch, projects]);
  const days = useMemo(
    () => datesBetween(validRange.from, validRange.to),
    [validRange],
  );
  const holidayMap = useMemo(
    () =>
      new Map(
        (portfolio?.holidays ?? []).map((holiday) => [holiday.date, holiday]),
      ),
    [portfolio?.holidays],
  );
  const timelineWidth = days.length * dayWidth;

  useEffect(() => {
    if (!portfolio) return;
    const next = autoRange(roleAdjustedRows);
    if (next.from !== validRange.from || next.to !== validRange.to) {
      setValidRange(next);
    }
  }, [portfolio, roleAdjustedRows, validRange.from, validRange.to]);

  function selectSavedFilterDraft(id: string) {
    setFilterSavedFilterDraft(id);
    if (id === systemFilterID) {
      setFilterProjectDraft(projects.map((project) => project.id));
      return;
    }
    const filter = filters.find((candidate) => candidate.id === id);
    const active = new Set(projects.map((project) => project.id));
    setFilterProjectDraft(
      (filter?.projectIds ?? []).filter((projectID) => active.has(projectID)),
    );
  }

  function openFilterDialog() {
    setProjectSearch("");
    setFilterError("");
    setFilterSavedFilterDraft(activeFilterID);
    setFilterProjectionDraft(projection);
    setFilterProjectDraft(selectedIDs);
    setFilterRoleDraft(effectiveRoleKeys);
    setDialog("filter");
  }

  function applyFilterDraft() {
    setActiveFilterID(filterSavedFilterDraft);
    setSelectedIDs([...filterProjectDraft].sort());
    setProjection(filterProjectionDraft);
    applyRoleDraft();
    setDialog(undefined);
  }

  function applyRoleDraft() {
    const allRoleKeys = roleOptions.map((option) => option.key).sort();
    const nextRoleKeys = [...filterRoleDraft].sort();
    if (sameStrings(allRoleKeys, nextRoleKeys)) {
      setRoleFilterActive(false);
      setSelectedRoleKeys([]);
      return;
    }
    setRoleFilterActive(true);
    setSelectedRoleKeys(nextRoleKeys);
  }

  async function saveFilter() {
    if (!draftSavedFilter || saving) return;
    setSaving(true);
    setFilterError("");
    try {
      const updated = await gateway.updateSavedFilter(
        draftSavedFilter.id,
        draftSavedFilter.version,
        filterProjectDraft,
      );
      setFilters((current) =>
        sortFilters(
          current.map((filter) =>
            filter.id === updated.id ? updated : filter,
          ),
        ),
      );
      setToast("Saved filter updated.");
    } catch (reason: unknown) {
      if (
        reason instanceof PortfolioOperationError &&
        reason.code === "SAVED_FILTER_CONFLICT"
      ) {
        setFilterError(
          "This saved filter changed elsewhere. Reloaded the latest version.",
        );
        setReloadMetadata((value) => value + 1);
      } else setFilterError(message(reason));
    } finally {
      setSaving(false);
    }
  }

  async function saveAs(event: FormEvent) {
    event.preventDefault();
    if (saving) return;
    setSaving(true);
    setFilterError("");
    try {
      const created = await gateway.createSavedFilter(
        filterName,
        filterProjectDraft,
      );
      setFilters((current) => sortFilters([...current, created]));
      setFilterSavedFilterDraft(created.id);
      setActiveFilterID(created.id);
      setSelectedIDs([...filterProjectDraft].sort());
      setProjection(filterProjectionDraft);
      applyRoleDraft();
      setDialog(undefined);
      setFilterName("");
      setToast("Saved filter created.");
    } catch (reason: unknown) {
      setFilterError(message(reason));
    } finally {
      setSaving(false);
    }
  }

  async function deleteFilter() {
    if (!draftSavedFilter || saving) return;
    setSaving(true);
    setFilterError("");
    try {
      await gateway.deleteSavedFilter(
        draftSavedFilter.id,
        draftSavedFilter.version,
      );
      setFilters((current) =>
        current.filter((filter) => filter.id !== draftSavedFilter.id),
      );
      setFilterSavedFilterDraft(systemFilterID);
      setActiveFilterID(systemFilterID);
      setSelectedIDs(projects.map((project) => project.id));
      setDialog(undefined);
      setToast("Saved filter deleted.");
    } catch (reason: unknown) {
      if (
        reason instanceof PortfolioOperationError &&
        reason.code === "SAVED_FILTER_CONFLICT"
      ) {
        setFilterError(
          "This saved filter changed elsewhere. Reload before deleting it.",
        );
        setReloadMetadata((value) => value + 1);
      } else setFilterError(message(reason));
    } finally {
      setSaving(false);
    }
  }

  async function mutateWBS(
    row: PortfolioRow,
    action: () => Promise<void>,
    successMessage: string,
    focusRetainedRow = false,
  ): Promise<boolean> {
    if (actionBusyRowID) return false;
    setActionBusyRowID(row.id);
    try {
      await action();
      setReloadProjection((value) => value + 1);
      if (focusRetainedRow)
        requestLocalRowFocus(row.id, currentScheduleProjectionVersion());
      setToast(successMessage);
      return true;
    } catch (reason: unknown) {
      setToast(
        reason instanceof WBSOperationError
          ? reason.message
          : reason instanceof Error
            ? reason.message
            : "The WBS item could not be updated.",
      );
      if (focusRetainedRow) requestLocalRowFocus(row.id);
      return false;
    } finally {
      setActionBusyRowID("");
    }
  }

  function requestLocalRowFocus(
    rowId: string,
    minimumProjectionVersion?: number,
  ) {
    const key = localFocusRequestVersion.current - 1;
    localFocusRequestVersion.current = key;
    setLocalFocusRequest({ key, rowId, minimumProjectionVersion });
  }

  async function confirmWBSDelete() {
    if (!pendingDelete) return;
    const selected = pendingDelete;
    const deleted = await mutateWBS(
      selected,
      () => wbsGateway.remove(selected.projectId, selected.id),
      "Task deleted.",
    );
    if (deleted) setPendingDelete(undefined);
  }

  return (
    <>
      <PageContent className="flex min-h-0 flex-1 flex-col">
        <section className="flex min-h-0 min-w-0 flex-1 flex-col">
          <section
            className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-surface border border-border-subtle bg-surface shadow-surface"
            aria-labelledby="portfolio-title"
          >
            <div className="flex items-center justify-between border-b border-border-subtle px-3 py-2">
              <h1
                id="portfolio-title"
                className="min-w-0 truncate text-base font-extrabold"
              >
                {portfolioTitle}
              </h1>
              <div className="flex shrink-0 items-center gap-2">
                {projectionLoading ? (
                  <span className="text-xs text-muted">Refreshing…</span>
                ) : null}
                <button
                  type="button"
                  className="grid size-8 place-items-center rounded-control border border-border-strong bg-surface text-text-secondary hover:border-brand hover:text-brand"
                  aria-label="Configure Gantt"
                  title="Configure Gantt"
                  onClick={openFilterDialog}
                >
                  <GanttConfigurationIcon />
                </button>
              </div>
            </div>
            {metadataLoading && projects.length === 0 ? (
              <ListSkeleton label="Loading Home" />
            ) : metadataError ? (
              <StateError
                message={metadataError}
                onRetry={() => setReloadMetadata((value) => value + 1)}
              />
            ) : projects.length === 0 ? (
              <EmptyState
                title="No active projects"
                description="Open or Locked projects will appear here. Saved filters remain available."
              />
            ) : selectedIDs.length === 0 ? (
              <EmptyState
                title="No projects selected"
                description="Select one or more active projects in Configure Gantt."
              />
            ) : projectionError ? (
              <StateError
                message={projectionError}
                onRetry={() => setReloadProjection((value) => value + 1)}
              />
            ) : !portfolio ||
              (projectionLoading && portfolio.rows.length === 0) ? (
              <ListSkeleton label="Loading portfolio timeline" />
            ) : (
              <Gantt
                rows={visibleRows}
                authoritativeRows={portfolio.rows}
                dependencies={portfolio.dependencies}
                holidays={portfolio.holidays}
                holidayMap={holidayMap}
                workingDayAnchor={portfolio.workingDayAnchor}
                days={days}
                timelineWidth={timelineWidth}
                collapsed={collapsed}
                restrictiveRoleFilter={restrictiveRoleFilter}
                busyRowID={actionBusyRowID}
                focusRequest={visibleFocusRequest}
                onFocusRequestHandled={(key) => {
                  if (localFocusRequest?.key === key)
                    setLocalFocusRequest(undefined);
                  else onFocusRequestHandled?.(key);
                }}
                onToggle={(id) =>
                  setCollapsed((current) => {
                    const next = new Set(current);
                    if (next.has(id)) next.delete(id);
                    else next.add(id);
                    return next;
                  })
                }
                onOpenProject={onOpenProject}
                onProjectCommand={onProjectCommand}
                onOpenWBS={onOpenWBS}
                onReorder={(row, direction) =>
                  void mutateWBS(
                    row,
                    () => wbsGateway.reorder(row.projectId, row.id, direction),
                    "Item reordered.",
                    true,
                  )
                }
                onDelete={setPendingDelete}
              />
            )}
          </section>
        </section>
      </PageContent>

      {dialog === "filter" ? (
        <Dialog
          titleID="portfolio-filter-title"
          onClose={() => setDialog(undefined)}
          wide
        >
          <form
            onSubmit={(event) => {
              event.preventDefault();
              applyFilterDraft();
            }}
            noValidate
          >
            <div>
              <h2
                id="portfolio-filter-title"
                className="text-dialog-title font-black"
              >
                Gantt configuration
              </h2>
              <p className="mt-1 text-sm text-muted">
                Choose the saved filter, projection, Projects, and Roles to
                display.
              </p>
            </div>

            <div className="mt-5 grid gap-4 rounded-control border border-border-subtle p-4 lg:grid-cols-[minmax(16rem,1fr)_auto]">
              <div className="min-w-0">
                <label
                  className="block text-sm font-extrabold"
                  htmlFor="portfolio-filter"
                >
                  Saved filter
                </label>
                <select
                  id="portfolio-filter"
                  className="mt-2 min-h-9 w-full rounded-control border border-border-strong bg-surface px-3 text-sm"
                  value={filterSavedFilterDraft}
                  onChange={(event) =>
                    selectSavedFilterDraft(event.target.value)
                  }
                >
                  <option value={systemFilterID}>All Active Projects</option>
                  {filters.map((filter) => (
                    <option key={filter.id} value={filter.id}>
                      {filter.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex flex-wrap items-end gap-2">
                <Button
                  type="button"
                  compact
                  disabled={!draftSavedFilter || saving}
                  onClick={() => void saveFilter()}
                >
                  Save
                </Button>
                <Button
                  type="button"
                  compact
                  disabled={saving}
                  onClick={() => {
                    setFilterError("");
                    setFilterName("");
                    setDialog("save-as");
                  }}
                >
                  Save As
                </Button>
                <Button
                  type="button"
                  compact
                  variant="danger"
                  disabled={!draftSavedFilter || saving}
                  onClick={() => {
                    setFilterError("");
                    setDialog("delete");
                  }}
                >
                  Delete
                </Button>
              </div>
            </div>
            {filterError ? (
              <p
                className="mt-2 text-sm font-semibold text-danger"
                role="alert"
              >
                {filterError}
              </p>
            ) : null}

            <fieldset className="mt-4 rounded-control border border-border-subtle p-4">
              <legend className="px-1 text-sm font-extrabold">
                Projection
              </legend>
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  role="switch"
                  aria-label="Execution projection"
                  aria-checked={filterProjectionDraft === "execution"}
                  variant={
                    filterProjectionDraft === "execution"
                      ? "primary"
                      : "secondary"
                  }
                  compact
                  onClick={() => setFilterProjectionDraft("execution")}
                >
                  Execution
                </Button>
                <Button
                  type="button"
                  role="switch"
                  aria-label="Commitment projection"
                  aria-checked={filterProjectionDraft === "commitment"}
                  variant={
                    filterProjectionDraft === "commitment"
                      ? "primary"
                      : "secondary"
                  }
                  compact
                  onClick={() => setFilterProjectionDraft("commitment")}
                >
                  Commitment
                </Button>
              </div>
            </fieldset>

            <div className="mt-5 grid gap-6 lg:grid-cols-2">
              <section aria-labelledby="project-filter-heading">
                <div className="flex items-center justify-between gap-3">
                  <h3 id="project-filter-heading" className="font-extrabold">
                    Projects
                  </h3>
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      compact
                      onClick={() =>
                        setFilterProjectDraft(
                          projects.map((project) => project.id),
                        )
                      }
                    >
                      Select All
                    </Button>
                    <Button
                      type="button"
                      compact
                      onClick={() => setFilterProjectDraft([])}
                    >
                      Clear All
                    </Button>
                  </div>
                </div>
                <input
                  className="mt-3 min-h-9 w-full rounded-control border border-border-strong px-3 text-sm"
                  type="search"
                  aria-label="Search projects"
                  placeholder="Search projects"
                  value={projectSearch}
                  onChange={(event) => setProjectSearch(event.target.value)}
                />
                <div
                  className="mt-3 max-h-80 space-y-2 overflow-auto"
                  role="group"
                  aria-label="Active projects"
                >
                  {visibleProjects.map((project) => (
                    <label
                      key={project.id}
                      className="flex min-w-0 items-center gap-2 rounded-control border border-border-subtle px-3 py-2 text-sm"
                    >
                      <input
                        type="checkbox"
                        checked={filterProjectDraftSet.has(project.id)}
                        onChange={(event) =>
                          setFilterProjectDraft((current) =>
                            event.target.checked
                              ? uniqueStrings([...current, project.id])
                              : current.filter((id) => id !== project.id),
                          )
                        }
                      />
                      <span className="min-w-0 truncate">{project.name}</span>
                      {project.status === "locked" ? (
                        <span className="rounded-action bg-surface-muted px-2 py-0.5 text-[10px] font-bold">
                          Locked
                        </span>
                      ) : null}
                    </label>
                  ))}
                </div>
              </section>

              <section aria-labelledby="role-filter-heading">
                <div className="flex items-center justify-between gap-3">
                  <h3 id="role-filter-heading" className="font-extrabold">
                    Roles
                  </h3>
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      compact
                      onClick={() =>
                        setFilterRoleDraft(
                          roleOptions.map((option) => option.key),
                        )
                      }
                    >
                      Select All
                    </Button>
                    <Button
                      type="button"
                      compact
                      onClick={() => setFilterRoleDraft([])}
                    >
                      Clear All
                    </Button>
                  </div>
                </div>
                <div
                  className="mt-3 max-h-80 space-y-2 overflow-auto"
                  role="group"
                  aria-label="Task roles"
                >
                  {roleOptions.length === 0 ? (
                    <p className="text-sm text-muted">
                      No Task roles are available in the selected Projects.
                    </p>
                  ) : (
                    roleOptions.map((role) => (
                      <label
                        key={role.key}
                        className="flex items-center gap-2 rounded-control border border-border-subtle px-3 py-2 text-sm"
                      >
                        <input
                          type="checkbox"
                          checked={filterRoleDraftSet.has(role.key)}
                          onChange={(event) =>
                            setFilterRoleDraft((current) =>
                              event.target.checked
                                ? uniqueStrings([...current, role.key])
                                : current.filter((key) => key !== role.key),
                            )
                          }
                        />
                        <span>{role.label}</span>
                      </label>
                    ))
                  )}
                </div>
              </section>
            </div>
            <div className="form-actions">
              <Button type="button" onClick={() => setDialog(undefined)}>
                Cancel
              </Button>
              <Button type="submit" variant="primary">
                Apply
              </Button>
            </div>
          </form>
        </Dialog>
      ) : null}

      {dialog === "save-as" ? (
        <Dialog
          titleID="save-filter-title"
          onClose={() => !saving && setDialog(undefined)}
        >
          <form onSubmit={(event) => void saveAs(event)} noValidate>
            <h2 id="save-filter-title" className="text-dialog-title font-black">
              Save filter as
            </h2>
            <FormField
              className="mt-5"
              id="saved-filter-name"
              name="name"
              label="Name"
              required
              data-autofocus
              maxLength={101}
              value={filterName}
              error={filterError}
              onChange={(event) => setFilterName(event.target.value)}
            />
            <div className="form-actions">
              <Button
                type="button"
                disabled={saving}
                onClick={() => setDialog(undefined)}
              >
                Cancel
              </Button>
              <Button type="submit" variant="primary" loading={saving}>
                Save As
              </Button>
            </div>
          </form>
        </Dialog>
      ) : null}
      {dialog === "delete" && draftSavedFilter ? (
        <Dialog
          titleID="delete-filter-title"
          kind="alertdialog"
          closeOnBackdrop={false}
          onClose={() => !saving && setDialog(undefined)}
        >
          <h2 id="delete-filter-title" className="text-dialog-title font-black">
            Delete {draftSavedFilter.name}?
          </h2>
          <p className="mt-3 text-muted">
            This removes the global saved filter. It does not change any
            project.
          </p>
          {filterError ? <Alert tone="danger">{filterError}</Alert> : null}
          <div className="form-actions">
            <Button disabled={saving} onClick={() => setDialog(undefined)}>
              Cancel
            </Button>
            <Button
              variant="danger-solid"
              loading={saving}
              onClick={() => void deleteFilter()}
            >
              Delete
            </Button>
          </div>
        </Dialog>
      ) : null}
      {pendingDelete ? (
        <Dialog
          titleID="home-wbs-delete-title"
          kind="alertdialog"
          onClose={() => !actionBusyRowID && setPendingDelete(undefined)}
        >
          <h2
            id="home-wbs-delete-title"
            className="text-dialog-title font-black"
          >
            Delete {pendingDelete.name}?
          </h2>
          <p className="mt-3 text-sm text-muted">
            This permanently removes the unfinished leaf Task.
          </p>
          <div className="form-actions">
            <Button
              disabled={Boolean(actionBusyRowID)}
              onClick={() => setPendingDelete(undefined)}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={Boolean(actionBusyRowID)}
              onClick={() => void confirmWBSDelete()}
            >
              Delete Task
            </Button>
          </div>
        </Dialog>
      ) : null}
      {toast ? <Toast message={toast} onDismiss={() => setToast("")} /> : null}
    </>
  );
}

function Gantt({
  rows,
  authoritativeRows,
  dependencies,
  holidays,
  holidayMap,
  workingDayAnchor,
  days,
  timelineWidth,
  collapsed,
  restrictiveRoleFilter,
  busyRowID,
  focusRequest,
  onFocusRequestHandled,
  onToggle,
  onOpenProject,
  onProjectCommand,
  onOpenWBS,
  onReorder,
  onDelete,
}: {
  rows: PortfolioRow[];
  authoritativeRows: PortfolioRow[];
  dependencies: PortfolioDependency[];
  holidays: PortfolioHoliday[];
  holidayMap: Map<string, PortfolioHoliday>;
  workingDayAnchor?: string;
  days: string[];
  timelineWidth: number;
  collapsed: Set<string>;
  restrictiveRoleFilter: boolean;
  busyRowID: string;
  focusRequest?: HomeRowFocusRequest;
  onFocusRequestHandled?(key: number): void;
  onToggle(id: string): void;
  onOpenProject(projectId: string): void;
  onProjectCommand(kind: HomeProjectCommandKind, projectId: string): void;
  onOpenWBS(request: WBSOpenRequest): void;
  onReorder(row: PortfolioRow, direction: "up" | "down"): void;
  onDelete(row: PortfolioRow): void;
}) {
  const leftBodyRef = useRef<HTMLDivElement>(null);
  const timelineHeaderRef = useRef<HTMLDivElement>(null);
  const timelineBodyRef = useRef<HTMLDivElement>(null);
  const rowNameRefs = useRef(new Map<string, HTMLButtonElement>());
  const handledFocusRequestKey = useRef<number | undefined>(undefined);
  const [columns, setColumns] = useState(initialColumns);
  const [resizingColumn, setResizingColumn] = useState<{
    index: number;
    startX: number;
    startWidth: number;
  }>();
  const [rowWindow, setRowWindow] = useState(() => ({
    start: 0,
    end: Math.min(rows.length, maximumRenderedRows),
  }));

  useEffect(() => {
    const timeline = timelineBodyRef.current;
    const viewportHeight = timeline?.clientHeight ?? 0;
    const currentScrollTop =
      timeline?.scrollTop ?? leftBodyRef.current?.scrollTop ?? 0;
    const retainedScrollTop = clamp(
      currentScrollTop,
      0,
      Math.max(0, rows.length * rowHeight - viewportHeight),
    );
    const nextWindow = rowWindowForViewport(
      rows.length,
      retainedScrollTop,
      viewportHeight,
    );

    if (timeline && timeline.scrollTop !== retainedScrollTop)
      timeline.scrollTop = retainedScrollTop;
    if (
      leftBodyRef.current &&
      leftBodyRef.current.scrollTop !== retainedScrollTop
    )
      leftBodyRef.current.scrollTop = retainedScrollTop;
    setRowWindow((current) =>
      current.start === nextWindow.start && current.end === nextWindow.end
        ? current
        : nextWindow,
    );
  }, [rows]);

  useEffect(() => {
    if (!focusRequest || handledFocusRequestKey.current === focusRequest.key)
      return;
    const targetIndex = rows.findIndex((row) => row.id === focusRequest.rowId);
    if (targetIndex < 0) return;
    const timeline = timelineBodyRef.current;
    const viewportHeight = timeline?.clientHeight ?? 0;
    const currentScrollTop =
      timeline?.scrollTop ?? leftBodyRef.current?.scrollTop ?? 0;
    const targetTop = targetIndex * rowHeight;
    const targetBottom = targetTop + rowHeight;
    const targetScrollTop =
      targetTop < currentScrollTop
        ? targetTop
        : targetBottom > currentScrollTop + viewportHeight
          ? Math.max(0, targetBottom - viewportHeight)
          : currentScrollTop;
    const nextWindow = rowWindowForViewport(
      rows.length,
      targetScrollTop,
      viewportHeight,
    );

    if (timeline) timeline.scrollTop = targetScrollTop;
    if (leftBodyRef.current) leftBodyRef.current.scrollTop = targetScrollTop;
    setRowWindow((current) =>
      current.start === nextWindow.start && current.end === nextWindow.end
        ? current
        : nextWindow,
    );
  }, [focusRequest, rows]);

  useEffect(() => {
    if (!focusRequest || handledFocusRequestKey.current === focusRequest.key)
      return;
    const target = rowNameRefs.current.get(focusRequest.rowId);
    if (!target) return;
    target.focus({ preventScroll: true });
    handledFocusRequestKey.current = focusRequest.key;
    onFocusRequestHandled?.(focusRequest.key);
  }, [focusRequest, onFocusRequestHandled, rowWindow, rows]);

  useEffect(() => {
    if (!resizingColumn) return;
    const activeResize = resizingColumn;
    function onMouseMove(event: MouseEvent) {
      setColumns((current) =>
        current.map((column, index) =>
          index === activeResize.index
            ? {
                ...column,
                width: clamp(
                  activeResize.startWidth + event.clientX - activeResize.startX,
                  column.minimum,
                  column.maximum,
                ),
              }
            : column,
        ),
      );
    }
    function onMouseUp() {
      setResizingColumn(undefined);
    }
    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onMouseUp);
    return () => {
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);
    };
  }, [resizingColumn]);

  const columnTemplate = columns.map((column) => `${column.width}px`).join(" ");
  const leftPaneWidth = columns.reduce(
    (total, column) => total + column.width,
    0,
  );
  const windowRows = rows.slice(rowWindow.start, rowWindow.end);
  const topSpacerHeight = rowWindow.start * rowHeight;
  const bottomSpacerHeight =
    Math.max(0, rows.length - rowWindow.end) * rowHeight;
  const rowIndex = new Map(rows.map((row, index) => [row.id, index]));
  const rowByID = new Map(rows.map((row) => [row.id, row]));
  const barFractions = useMemo(
    () => sameDayDependencyFractions(rows, dependencies),
    [dependencies, rows],
  );
  const projectStatus = new Map(
    rows
      .filter((row) => row.kind === "project")
      .map((row) => [row.projectId, row.status]),
  );
  const from = days[0];
  const to = days.at(-1);
  const arrows = dependencies.flatMap((dependency) => {
    const blocking = rowByID.get(dependency.blockingTaskId);
    const blocked = rowByID.get(dependency.blockedTaskId);
    if (
      !blocking?.end ||
      !blocked?.start ||
      !from ||
      !to ||
      blocking.end < from ||
      blocking.end > to ||
      blocked.start < from ||
      blocked.start > to
    )
      return [];
    const blockingIndex = rowIndex.get(blocking.id);
    const blockedIndex = rowIndex.get(blocked.id);
    if (blockingIndex === undefined || blockedIndex === undefined) return [];
    if (
      blockingIndex < rowWindow.start ||
      blockingIndex >= rowWindow.end ||
      blockedIndex < rowWindow.start ||
      blockedIndex >= rowWindow.end
    )
      return [];
    return [
      {
        ...dependency,
        x1: barEndAnchor(
          blocking.end,
          from,
          barFractions.get(blocking.id)?.end,
        ),
        y1: blockingIndex * rowHeight + rowHeight / 2,
        x2: barStartAnchor(
          blocked.start,
          from,
          barFractions.get(blocked.id)?.start,
        ),
        y2: blockedIndex * rowHeight + rowHeight / 2,
      },
    ];
  });
  const nonWorkingBackground = nonWorkingGradient(days, holidayMap);
  const months = monthSegments(days);

  function updateRowWindow(scrollTop: number, clientHeight: number) {
    const next = rowWindowForViewport(rows.length, scrollTop, clientHeight);
    setRowWindow((current) =>
      current.start === next.start && current.end === next.end ? current : next,
    );
  }

  return (
    <div
      className="grid min-h-0 flex-1 overflow-hidden"
      style={{
        gridTemplateColumns: `${leftPaneWidth}px minmax(0, 1fr)`,
        gridTemplateRows: `${ganttHeaderHeight}px minmax(0, 1fr)`,
      }}
    >
      <div
        className="grid border-r border-b border-border-strong bg-surface text-[11px] font-extrabold"
        style={{ gridTemplateColumns: columnTemplate }}
      >
        {columns.map((column, index) => (
          <div
            key={column.key}
            className="relative flex min-w-0 items-center border-r border-border-subtle px-2"
          >
            <span className="truncate">{column.label}</span>
            <div
              role="separator"
              aria-label={`Resize ${column.label} column`}
              aria-orientation="vertical"
              aria-valuemin={column.minimum}
              aria-valuemax={column.maximum}
              aria-valuenow={column.width}
              tabIndex={0}
              className="absolute inset-y-0 right-0 z-10 w-2 translate-x-1/2 cursor-col-resize focus:bg-brand-soft focus:outline-none"
              onMouseDown={(event) =>
                setResizingColumn({
                  index,
                  startX: event.clientX,
                  startWidth: column.width,
                })
              }
              onKeyDown={(event) => {
                if (event.key !== "ArrowLeft" && event.key !== "ArrowRight")
                  return;
                event.preventDefault();
                const delta = event.key === "ArrowLeft" ? -12 : 12;
                setColumns((current) =>
                  current.map((candidate, candidateIndex) =>
                    candidateIndex === index
                      ? {
                          ...candidate,
                          width: clamp(
                            candidate.width + delta,
                            candidate.minimum,
                            candidate.maximum,
                          ),
                        }
                      : candidate,
                  ),
                );
              }}
            />
          </div>
        ))}
      </div>

      <div
        ref={timelineHeaderRef}
        className="overflow-hidden border-b border-border-strong bg-surface"
      >
        <div style={{ width: timelineWidth }}>
          <div className="flex h-6 border-b border-border-subtle">
            {days.map((date) => (
              <div
                key={`sequence-${date}`}
                className="grid shrink-0 place-items-center border-r border-border-subtle text-[10px] font-bold"
                style={{ width: dayWidth }}
              >
                {workingDayNumber(date, workingDayAnchor, holidayMap)}
              </div>
            ))}
          </div>
          <div
            className="flex h-6 border-b border-border-subtle"
            role="group"
            aria-label="Timeline months"
          >
            {months.map((month) => (
              <div
                key={month.key}
                className="grid shrink-0 place-items-center border-r border-border-subtle text-[10px] font-extrabold"
                style={{ width: month.dayCount * dayWidth }}
              >
                {month.label}
              </div>
            ))}
          </div>
          <div className="flex h-9">
            {days.map((date) => {
              const holiday = holidayMap.get(date);
              const nonWorking = isWeekend(date) || Boolean(holiday);
              return (
                <div
                  key={date}
                  className={`grid shrink-0 place-items-center border-r border-border-subtle text-center text-[10px] leading-3 ${nonWorking ? "bg-surface-muted" : ""}`}
                  style={{ width: dayWidth }}
                  title={holiday?.description}
                >
                  <span aria-hidden="true">{shortWeekday(date)}</span>
                  <span aria-hidden="true">{dayOfMonth(date)}</span>
                  <span className="sr-only">
                    {holiday
                      ? `${date}, public holiday: ${holiday.description}`
                      : date}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div
        ref={leftBodyRef}
        className="overflow-hidden border-r border-border-strong bg-surface"
        onWheel={(event) => {
          const timeline = timelineBodyRef.current;
          if (!timeline) return;
          timeline.scrollTop += event.deltaY;
        }}
      >
        <div
          role="treegrid"
          aria-label="Portfolio schedule rows"
          aria-rowcount={rows.length}
          aria-colcount={columns.length}
          style={{ width: leftPaneWidth }}
        >
          <div
            role="rowgroup"
            style={{
              paddingTop: topSpacerHeight,
              paddingBottom: bottomSpacerHeight,
            }}
          >
            {windowRows.map((row, windowIndex) => (
              <div
                key={row.id}
                className="grid border-b border-border-subtle text-xs"
                role="row"
                aria-rowindex={rowWindow.start + windowIndex + 1}
                aria-level={row.depth + 1}
                aria-expanded={
                  row.hasChildren ? !collapsed.has(row.id) : undefined
                }
                data-portfolio-row={row.id}
                style={{
                  height: rowHeight,
                  gridTemplateColumns: columnTemplate,
                }}
              >
                <GridCell>
                  {row.kind === "project" ? "" : row.wbsNumber}
                </GridCell>
                <div
                  className="group relative flex min-w-0 items-center gap-1 overflow-hidden border-r border-border-subtle px-2"
                  role="gridcell"
                  style={{
                    paddingLeft: `${Math.min(row.depth, 8) * 10 + 6}px`,
                  }}
                >
                  {row.hasChildren ? (
                    <button
                      type="button"
                      className="grid size-6 shrink-0 place-items-center rounded-action hover:bg-surface-muted"
                      aria-label={`${collapsed.has(row.id) ? "Expand" : "Collapse"} ${row.name}`}
                      aria-expanded={!collapsed.has(row.id)}
                      onClick={() => onToggle(row.id)}
                    >
                      {collapsed.has(row.id) ? "▸" : "▾"}
                    </button>
                  ) : (
                    <span className="w-5 shrink-0" />
                  )}
                  <button
                    ref={(element) => {
                      if (element) rowNameRefs.current.set(row.id, element);
                      else rowNameRefs.current.delete(row.id);
                    }}
                    type="button"
                    className="min-w-0 truncate text-left font-semibold hover:text-brand-strong hover:underline"
                    title={row.name}
                    onClick={() =>
                      row.kind === "project"
                        ? onOpenProject(row.projectId)
                        : onOpenWBS({
                            projectId: row.projectId,
                            nodeId: row.id,
                          })
                    }
                  >
                    {row.name}
                  </button>
                  {row.status === "locked" ? (
                    <span className="rounded-action bg-surface-muted px-1.5 py-0.5 text-[10px] font-bold">
                      Locked
                    </span>
                  ) : null}
                  {row.incompleteEffort ? (
                    <span
                      role="img"
                      title="Contains task without effort"
                      aria-label="Contains task without effort"
                    >
                      !
                    </span>
                  ) : null}
                  {row.incompleteSchedule ? (
                    <span
                      role="img"
                      title="Contains unscheduled task"
                      aria-label="Contains unscheduled task"
                    >
                      ◌
                    </span>
                  ) : null}
                  <RowActions
                    row={row}
                    authoritativeRows={authoritativeRows}
                    projectStatus={projectStatus.get(row.projectId)}
                    restrictiveRoleFilter={restrictiveRoleFilter}
                    busy={Boolean(busyRowID)}
                    onProjectCommand={onProjectCommand}
                    onOpenWBS={onOpenWBS}
                    onReorder={onReorder}
                    onDelete={onDelete}
                  />
                </div>
                <GridCell>
                  {row.kind === "task" ? (row.roleName ?? "") : ""}
                </GridCell>
                <GridCell>{row.assigneeName ?? ""}</GridCell>
                <GridCell>
                  {row.effortMinutes === undefined
                    ? ""
                    : formatEffort(row.effortMinutes)}
                </GridCell>
                <GridCell>{formatDisplayDate(row.start)}</GridCell>
                <GridCell>{formatDisplayDate(row.end)}</GridCell>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div
        ref={timelineBodyRef}
        className="overflow-auto"
        role="region"
        tabIndex={0}
        aria-label="Scrollable portfolio timeline"
        onScroll={(event) => {
          const viewport = event.currentTarget;
          if (
            leftBodyRef.current &&
            leftBodyRef.current.scrollTop !== viewport.scrollTop
          ) {
            leftBodyRef.current.scrollTop = viewport.scrollTop;
          }
          if (
            timelineHeaderRef.current &&
            timelineHeaderRef.current.scrollLeft !== viewport.scrollLeft
          ) {
            timelineHeaderRef.current.scrollLeft = viewport.scrollLeft;
          }
          updateRowWindow(viewport.scrollTop, viewport.clientHeight);
        }}
      >
        <div
          className="relative"
          style={{
            width: timelineWidth,
            minHeight: rows.length * rowHeight,
          }}
        >
          <div
            style={{
              paddingTop: topSpacerHeight,
              paddingBottom: bottomSpacerHeight,
            }}
          >
            {windowRows.map((row) => (
              <div
                key={row.id}
                className="border-b border-border-subtle"
                style={{ height: rowHeight }}
              >
                <div
                  className="relative h-full w-full overflow-hidden text-left"
                  style={{ backgroundImage: nonWorkingBackground }}
                  role="img"
                  aria-label={`${row.name}: ${
                    row.start && row.end
                      ? `${row.start} through ${row.end}`
                      : "unscheduled"
                  }`}
                >
                  {row.start &&
                  row.end &&
                  from &&
                  to &&
                  row.end >= from &&
                  row.start <= to ? (
                    <span
                      className={`absolute top-2 h-6 rounded-action border ${
                        row.kind === "task"
                          ? "border-brand bg-brand/80"
                          : "border-brand-dark bg-brand-soft"
                      }`}
                      style={barStyle(
                        row.start,
                        row.end,
                        from,
                        days.length,
                        barFractions.get(row.id),
                      )}
                      title={`${row.name}: ${row.start} – ${row.end}`}
                    />
                  ) : null}
                </div>
              </div>
            ))}
          </div>
          <svg
            className="pointer-events-none absolute inset-0"
            style={{ width: timelineWidth, height: rows.length * rowHeight }}
            aria-hidden="true"
          >
            <defs>
              <marker
                id="portfolio-arrow"
                markerWidth="7"
                markerHeight="7"
                refX="6"
                refY="3.5"
                orient="auto"
              >
                <path d="M0,0 L7,3.5 L0,7 z" fill="currentColor" />
              </marker>
            </defs>
            {arrows.map((arrow) => {
              return (
                <path
                  key={arrow.id}
                  data-portfolio-dependency={arrow.id}
                  d={dependencyPath(arrow.x1, arrow.y1, arrow.x2, arrow.y2)}
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  markerEnd="url(#portfolio-arrow)"
                />
              );
            })}
          </svg>
        </div>
        {rows.length === 0 ? (
          <div className="state-panel">
            <p className="font-bold">
              No Tasks match the selected Project and Role filters.
            </p>
          </div>
        ) : null}
        <ul className="sr-only" aria-label="Visible dependencies">
          {arrows.map((arrow) => (
            <li key={`text-${arrow.id}`}>
              {arrow.blockingTaskId} blocks {arrow.blockedTaskId}
            </li>
          ))}
        </ul>
        <span className="sr-only">
          {holidays.length} public holiday dates are represented in the visible
          range.
        </span>
      </div>
    </div>
  );
}

function GanttConfigurationIcon() {
  return (
    <svg
      className="size-4"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 7h10M18 7h2M4 17h2M10 17h10" />
      <circle cx="16" cy="7" r="2" />
      <circle cx="8" cy="17" r="2" />
    </svg>
  );
}

type RowAction = {
  value: string;
  label: string;
  disabled?: boolean;
  disabledReason?: string;
  destructive?: boolean;
};

function RowActions({
  row,
  authoritativeRows,
  projectStatus,
  restrictiveRoleFilter,
  busy,
  onProjectCommand,
  onOpenWBS,
  onReorder,
  onDelete,
}: {
  row: PortfolioRow;
  authoritativeRows: PortfolioRow[];
  projectStatus?: "open" | "locked";
  restrictiveRoleFilter: boolean;
  busy: boolean;
  onProjectCommand(kind: HomeProjectCommandKind, projectId: string): void;
  onOpenWBS(request: WBSOpenRequest): void;
  onReorder(row: PortfolioRow, direction: "up" | "down"): void;
  onDelete(row: PortfolioRow): void;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [menuPosition, setMenuPosition] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const open = projectStatus === "open";
  const quickAction =
    row.kind === "project" && row.status === "open"
      ? "task"
      : row.kind !== "project" && open && !row.completed
        ? "child"
        : undefined;
  const siblings = authoritativeRows
    .filter(
      (candidate) =>
        candidate.kind !== "project" &&
        candidate.projectId === row.projectId &&
        candidate.parentId === row.parentId,
    )
    .sort((left, right) => left.position - right.position);
  const siblingIndex = siblings.findIndex(
    (candidate) => candidate.id === row.id,
  );
  const actions: RowAction[] = [];

  if (row.kind === "project") {
    if (row.status === "open") {
      actions.push({ value: "lock", label: "Lock" });
      actions.push({ value: "close", label: "Close" });
      if (!row.hasChildren)
        actions.push({
          value: "delete-project",
          label: "Delete",
          destructive: true,
        });
    } else if (row.status === "locked") {
      actions.push({ value: "reopen", label: "Reopen" });
      actions.push({ value: "close", label: "Close" });
    }
  } else if (open) {
    actions.push({
      value: "move-up",
      label: "Move Up",
      disabled: restrictiveRoleFilter || siblingIndex <= 0,
      disabledReason: restrictiveRoleFilter
        ? "Show all roles to reorder WBS items."
        : "This item is already the first sibling.",
    });
    actions.push({
      value: "move-down",
      label: "Move Down",
      disabled:
        restrictiveRoleFilter ||
        siblingIndex < 0 ||
        siblingIndex === siblings.length - 1,
      disabledReason: restrictiveRoleFilter
        ? "Show all roles to reorder WBS items."
        : "This item is already the last sibling.",
    });
    actions.push({ value: "move-to", label: "Move to…" });
    if (row.kind === "task" && !row.completed && !row.hasChildren)
      actions.push({
        value: "delete-task",
        label: "Delete",
        destructive: true,
      });
  }

  useEffect(() => {
    if (!menuOpen) return;
    const menu = menuRef.current;
    const firstAction =
      menu?.querySelector<HTMLButtonElement>('[role="menuitem"]');
    firstAction?.focus();

    function closeOnOutsideInteraction(event: Event) {
      const target = event.target;
      if (!(target instanceof Node)) return;
      if (menuRef.current?.contains(target)) return;
      if (triggerRef.current?.contains(target)) return;
      setMenuOpen(false);
    }
    function closeForViewportChange() {
      setMenuOpen(false);
    }
    document.addEventListener("pointerdown", closeOnOutsideInteraction);
    document.addEventListener("focusin", closeOnOutsideInteraction);
    window.addEventListener("resize", closeForViewportChange);
    window.addEventListener("scroll", closeForViewportChange, true);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsideInteraction);
      document.removeEventListener("focusin", closeOnOutsideInteraction);
      window.removeEventListener("resize", closeForViewportChange);
      window.removeEventListener("scroll", closeForViewportChange, true);
    };
  }, [menuOpen]);

  function toggleMenu() {
    if (menuOpen) {
      setMenuOpen(false);
      return;
    }
    const trigger = triggerRef.current;
    if (!trigger) return;
    const bounds = trigger.getBoundingClientRect();
    const width = 160;
    const estimatedHeight = actions.length * 36 + 8;
    const below = bounds.bottom + 4;
    setMenuPosition({
      top:
        below + estimatedHeight <= window.innerHeight
          ? below
          : Math.max(8, bounds.top - estimatedHeight - 4),
      left: Math.max(
        8,
        Math.min(bounds.right - width, window.innerWidth - width - 8),
      ),
    });
    setMenuOpen(true);
  }

  function closeMenuAndFocus() {
    setMenuOpen(false);
    triggerRef.current?.focus();
  }

  function invoke(value: string) {
    closeMenuAndFocus();
    if (value === "lock" || value === "close" || value === "reopen")
      onProjectCommand(value, row.projectId);
    else if (value === "delete-project")
      onProjectCommand("delete", row.projectId);
    else if (value === "move-up") onReorder(row, "up");
    else if (value === "move-down") onReorder(row, "down");
    else if (value === "move-to")
      onOpenWBS({ projectId: row.projectId, moveNodeId: row.id });
    else if (value === "delete-task") onDelete(row);
  }

  if (!quickAction && actions.length === 0) return null;

  return (
    <div
      className={`absolute inset-y-0 right-1 z-20 flex items-center justify-end gap-1 bg-surface pl-1 transition-opacity ${
        menuOpen
          ? "pointer-events-auto opacity-100"
          : "pointer-events-none opacity-0 group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100"
      }`}
      data-row-actions={row.id}
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          event.stopPropagation();
          closeMenuAndFocus();
        }
      }}
    >
      {quickAction ? (
        <button
          type="button"
          className="grid size-7 place-items-center rounded-action text-brand-strong hover:bg-brand-soft disabled:opacity-50"
          aria-label={`${quickAction === "task" ? "Add Task" : "Add Child"} for ${row.name}`}
          title={`${quickAction === "task" ? "Add Task to" : "Add Child under"} ${row.name}`}
          disabled={busy}
          onClick={() =>
            onOpenWBS({
              projectId: row.projectId,
              createParentId: quickAction === "task" ? null : row.id,
            })
          }
        >
          {quickAction === "task" ? <AddTaskIcon /> : <AddChildIcon />}
        </button>
      ) : null}
      {actions.length > 0 ? (
        <>
          <button
            ref={triggerRef}
            type="button"
            className="grid size-7 place-items-center rounded-action border border-transparent bg-surface text-base font-black hover:border-border-strong disabled:opacity-50"
            aria-label={`More actions for ${row.name}`}
            aria-haspopup="menu"
            aria-expanded={menuOpen}
            title={`More actions for ${row.name}`}
            disabled={busy}
            onClick={toggleMenu}
          >
            ⋯
          </button>
          {menuOpen
            ? createPortal(
                <div
                  ref={menuRef}
                  className="fixed z-[100] min-w-40 rounded-control border border-border-strong bg-surface p-1 shadow-surface"
                  role="menu"
                  aria-label={`Actions for ${row.name}`}
                  style={menuPosition}
                  onKeyDown={(event) => {
                    if (event.key === "Escape") {
                      event.stopPropagation();
                      closeMenuAndFocus();
                    }
                  }}
                >
                  {actions.map((action) => {
                    const blocked = Boolean(action.disabled || busy);
                    return (
                      <button
                        key={action.value}
                        type="button"
                        role="menuitem"
                        aria-disabled={blocked}
                        className={`block w-full rounded-action px-3 py-2 text-left text-xs font-semibold hover:bg-surface-muted ${
                          blocked ? "cursor-not-allowed opacity-50" : ""
                        } ${
                          action.destructive
                            ? "mt-1 border-t border-border-subtle text-danger"
                            : ""
                        }`}
                        title={
                          action.disabled ? action.disabledReason : undefined
                        }
                        onClick={() => {
                          if (!blocked) invoke(action.value);
                        }}
                      >
                        {action.label}
                        {action.disabled && action.disabledReason ? (
                          <span className="sr-only">
                            {" "}
                            {action.disabledReason}
                          </span>
                        ) : null}
                      </button>
                    );
                  })}
                </div>,
                document.body,
              )
            : null}
        </>
      ) : null}
    </div>
  );
}

function AddTaskIcon() {
  return (
    <svg
      className="size-4"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M12 8v8M8 12h8" />
    </svg>
  );
}

function AddChildIcon() {
  return (
    <svg
      className="size-4"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M6 4v8a4 4 0 0 0 4 4h4" />
      <path d="M14 12v8M10 16h8" />
    </svg>
  );
}

function GridCell({ children }: { children: string }) {
  return (
    <div
      className="flex min-w-0 items-center overflow-hidden border-r border-border-subtle px-2 text-[11px]"
      role="gridcell"
      title={children}
    >
      <span className="truncate">{children}</span>
    </div>
  );
}

function StateError({
  message,
  onRetry,
}: {
  message: string;
  onRetry(): void;
}) {
  return (
    <div className="state-panel">
      <p className="font-bold text-danger" role="alert">
        {message}
      </p>
      <Button className="mt-4" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function collectRoleOptions(rows: PortfolioRow[]): RoleOption[] {
  const values = new Map<string, string>();
  for (const row of rows) {
    if (row.kind !== "task") continue;
    const key = row.roleId ?? unassignedRoleKey;
    values.set(key, row.roleName ?? "No role");
  }
  return [...values.entries()]
    .map(([key, label]) => ({ key, label }))
    .sort(
      (left, right) =>
        left.label.localeCompare(right.label, undefined, {
          sensitivity: "base",
        }) || left.key.localeCompare(right.key),
    );
}

function applyRoleFilter(
  rows: PortfolioRow[],
  selectedRoleKeys: Set<string>,
  active: boolean,
): PortfolioRow[] {
  const byID = new Map(rows.map((row) => [row.id, row]));
  const matchingTasks = rows.filter(
    (row) =>
      row.kind === "task" &&
      (!active || selectedRoleKeys.has(row.roleId ?? unassignedRoleKey)),
  );
  const included = new Set<string>();
  for (const task of matchingTasks) {
    included.add(task.id);
    included.add(task.projectId);
    let parentID = task.parentId;
    while (parentID) {
      included.add(parentID);
      parentID = byID.get(parentID)?.parentId;
    }
  }

  return rows.flatMap((row) => {
    if (row.kind === "task") {
      return included.has(row.id) ? [row] : [];
    }
    if (active && !included.has(row.id)) return [];
    const descendants = matchingTasks.filter((task) =>
      isDescendantOf(task, row, byID),
    );
    if (descendants.length === 0) return active ? [] : [row];
    const starts = descendants.flatMap((task) =>
      task.start && task.end ? [task.start] : [],
    );
    const ends = descendants.flatMap((task) =>
      task.start && task.end ? [task.end] : [],
    );
    const knownEffort = descendants.flatMap((task) =>
      task.effortMinutes === undefined ? [] : [task.effortMinutes],
    );
    return [
      {
        ...row,
        roleId: undefined,
        roleName: undefined,
        effortMinutes:
          knownEffort.length === 0
            ? undefined
            : knownEffort.reduce((total, value) => total + value, 0),
        start: starts.length === 0 ? undefined : starts.sort()[0],
        end: ends.length === 0 ? undefined : ends.sort().at(-1),
        incompleteEffort: descendants.some(
          (task) => task.effortMinutes === undefined,
        ),
        incompleteSchedule: descendants.some(
          (task) => !task.start || !task.end || task.unscheduledReason,
        ),
        hasChildren: descendants.length > 0,
      },
    ];
  });
}

function isDescendantOf(
  task: PortfolioRow,
  ancestor: PortfolioRow,
  byID: Map<string, PortfolioRow>,
): boolean {
  if (ancestor.kind === "project") return task.projectId === ancestor.projectId;
  let parentID = task.parentId;
  while (parentID) {
    if (parentID === ancestor.id) return true;
    parentID = byID.get(parentID)?.parentId;
  }
  return false;
}

function visiblePortfolioRows(
  rows: PortfolioRow[],
  collapsed: Set<string>,
): PortfolioRow[] {
  const visible: PortfolioRow[] = [];
  const collapsedDepths: number[] = [];
  for (const row of rows) {
    while (collapsedDepths.length > 0 && row.depth <= collapsedDepths.at(-1)!)
      collapsedDepths.pop();
    if (collapsedDepths.length === 0) visible.push(row);
    if (collapsed.has(row.id)) collapsedDepths.push(row.depth);
  }
  return visible;
}

function focusRequestForRoleAdjustedRows(
  request: HomeRowFocusRequest | undefined,
  authoritativeRows: PortfolioRow[],
  roleAdjustedRows: PortfolioRow[],
): HomeRowFocusRequest | undefined {
  if (!request) return undefined;
  const target = authoritativeRows.find((row) => row.id === request.rowId);
  if (!target) return request;
  const includedIDs = new Set(roleAdjustedRows.map((row) => row.id));
  if (includedIDs.has(target.id)) return request;

  const byID = new Map(authoritativeRows.map((row) => [row.id, row]));
  let ancestorID = target.parentId;
  while (ancestorID) {
    if (includedIDs.has(ancestorID)) return { ...request, rowId: ancestorID };
    ancestorID = byID.get(ancestorID)?.parentId;
  }
  const project = authoritativeRows.find(
    (row) => row.kind === "project" && row.projectId === target.projectId,
  );
  return project && includedIDs.has(project.id)
    ? { ...request, rowId: project.id }
    : request;
}

function autoRange(rows: PortfolioRow[]): ValidRange {
  const taskDates = rows.filter(
    (row) => row.kind === "task" && row.start && row.end,
  );
  if (taskDates.length === 0) return currentMonthRange();
  const starts = taskDates.map((row) => row.start!).sort();
  const ends = taskDates.map((row) => row.end!).sort();
  return { from: starts[0], to: addDays(ends.at(-1)!, 7) };
}

function portfolioQueryRange(): ValidRange {
  const today = formatISODate(new Date());
  return {
    from: addDays(today, -364),
    to: addDays(today, 365),
  };
}
function currentMonthRange(): ValidRange {
  const now = new Date();
  const from = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));
  const to = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 0));
  return { from: formatISODate(from), to: formatISODate(to) };
}

function datesBetween(from: string, to: string): string[] {
  const values: string[] = [];
  const cursor = parseISODate(from);
  const end = parseISODate(to);
  while (cursor <= end) {
    values.push(formatISODate(cursor));
    cursor.setUTCDate(cursor.getUTCDate() + 1);
  }
  return values;
}

function workingDayNumber(
  date: string,
  anchor: string | undefined,
  holidays: Map<string, PortfolioHoliday>,
): string {
  if (!anchor || date < anchor || isWeekend(date) || holidays.has(date))
    return "";
  let count = 0;
  const cursor = parseISODate(anchor);
  const end = parseISODate(date);
  while (cursor <= end) {
    const value = formatISODate(cursor);
    if (!isWeekend(value) && !holidays.has(value)) count += 1;
    cursor.setUTCDate(cursor.getUTCDate() + 1);
  }
  return String(count);
}

function nonWorkingGradient(
  days: string[],
  holidays: Map<string, PortfolioHoliday>,
): string {
  const stripes = days.flatMap((date, index) => {
    if (!isWeekend(date) && !holidays.has(date)) return [];
    const start = index * dayWidth;
    return [
      `transparent ${start}px`,
      `var(--color-surface-muted) ${start}px`,
      `var(--color-surface-muted) ${start + dayWidth}px`,
      `transparent ${start + dayWidth}px`,
    ];
  });
  return stripes.length === 0
    ? "none"
    : `linear-gradient(to right, ${stripes.join(",")})`;
}

function barStyle(
  start: string,
  end: string,
  from: string,
  dayCount: number,
  fractions?: BarFractions,
): CSSProperties {
  const leftDays = Math.max(0, daysBetweenISO(from, start));
  const endDays = Math.min(dayCount - 1, daysBetweenISO(from, end));
  const startFraction = start < from ? 0 : (fractions?.start ?? 0);
  const visibleTo = addDays(from, dayCount - 1);
  const endFraction = end > visibleTo ? 1 : (fractions?.end ?? 1);
  const left = leftDays * dayWidth + clamp(startFraction, 0, 1) * dayWidth + 3;
  const right = endDays * dayWidth + clamp(endFraction, 0, 1) * dayWidth - 3;
  return {
    left,
    width: Math.max(2, right - left),
  };
}

function barStartAnchor(date: string, from: string, fraction = 0): number {
  return (
    daysBetweenISO(from, date) * dayWidth + clamp(fraction, 0, 1) * dayWidth + 3
  );
}

function barEndAnchor(date: string, from: string, fraction = 1): number {
  return (
    daysBetweenISO(from, date) * dayWidth + clamp(fraction, 0, 1) * dayWidth - 3
  );
}

function dependencyPath(
  x1: number,
  y1: number,
  x2: number,
  y2: number,
): string {
  if (x2 >= x1) {
    const middle = x1 + (x2 - x1) / 2;
    return `M ${x1} ${y1} H ${middle} V ${y2} H ${x2}`;
  }
  const departure = x1 + 8;
  const approach = Math.max(0, x2 - 8);
  return `M ${x1} ${y1} H ${departure} V ${y2} H ${approach} H ${x2}`;
}

function sameDayDependencyFractions(
  rows: PortfolioRow[],
  dependencies: PortfolioDependency[],
): Map<string, BarFractions> {
  // Equal slots communicate order only. They deliberately do not claim exact
  // intraday allocation duration, which is outside the Home projection.
  const rowByID = new Map(rows.map((row) => [row.id, row]));
  const edgesByDate = new Map<
    string,
    { blockingTaskId: string; blockedTaskId: string }[]
  >();
  for (const dependency of dependencies) {
    const blocking = rowByID.get(dependency.blockingTaskId);
    const blocked = rowByID.get(dependency.blockedTaskId);
    if (
      blocking?.kind !== "task" ||
      blocked?.kind !== "task" ||
      !blocking.end ||
      !blocked.start ||
      blocking.end !== blocked.start ||
      !blocking.assigneeId ||
      blocking.assigneeId !== blocked.assigneeId ||
      blocking.id === blocked.id
    )
      continue;
    const edges = edgesByDate.get(blocking.end) ?? [];
    edges.push({
      blockingTaskId: blocking.id,
      blockedTaskId: blocked.id,
    });
    edgesByDate.set(blocking.end, edges);
  }

  const result = new Map<string, BarFractions>();
  for (const [date, edges] of edgesByDate) {
    const outgoing = new Map<string, string[]>();
    const incoming = new Map<string, string[]>();
    const connected = new Map<string, Set<string>>();
    for (const edge of edges) {
      outgoing.set(edge.blockingTaskId, [
        ...(outgoing.get(edge.blockingTaskId) ?? []),
        edge.blockedTaskId,
      ]);
      incoming.set(edge.blockedTaskId, [
        ...(incoming.get(edge.blockedTaskId) ?? []),
        edge.blockingTaskId,
      ]);
      const blockingConnected =
        connected.get(edge.blockingTaskId) ?? new Set<string>();
      blockingConnected.add(edge.blockedTaskId);
      connected.set(edge.blockingTaskId, blockingConnected);
      const blockedConnected =
        connected.get(edge.blockedTaskId) ?? new Set<string>();
      blockedConnected.add(edge.blockingTaskId);
      connected.set(edge.blockedTaskId, blockedConnected);
    }

    const remaining = new Set(connected.keys());
    while (remaining.size > 0) {
      const first = remaining.values().next().value as string;
      const component = new Set<string>();
      const stack = [first];
      while (stack.length > 0) {
        const id = stack.pop();
        if (!id || component.has(id)) continue;
        component.add(id);
        remaining.delete(id);
        for (const adjacent of connected.get(id) ?? []) stack.push(adjacent);
      }

      const indegree = new Map<string, number>();
      const rank = new Map<string, number>();
      for (const id of component) {
        indegree.set(
          id,
          (incoming.get(id) ?? []).filter((value) => component.has(value))
            .length,
        );
        rank.set(id, 0);
      }
      const queue = [...component]
        .filter((id) => indegree.get(id) === 0)
        .sort();
      let processed = 0;
      while (queue.length > 0) {
        const id = queue.shift();
        if (!id) continue;
        processed += 1;
        for (const next of outgoing.get(id) ?? []) {
          if (!component.has(next)) continue;
          rank.set(
            next,
            Math.max(rank.get(next) ?? 0, (rank.get(id) ?? 0) + 1),
          );
          const nextIndegree = (indegree.get(next) ?? 0) - 1;
          indegree.set(next, nextIndegree);
          if (nextIndegree === 0) {
            queue.push(next);
            queue.sort();
          }
        }
      }
      if (processed !== component.size) continue;

      const slotCount =
        Math.max(...[...component].map((id) => rank.get(id) ?? 0)) + 1;
      for (const id of component) {
        const row = rowByID.get(id);
        if (!row) continue;
        const current = result.get(id) ?? { start: 0, end: 1 };
        const slot = rank.get(id) ?? 0;
        if ((incoming.get(id)?.length ?? 0) > 0 && row.start === date) {
          current.start = Math.max(current.start, slot / slotCount);
        }
        if ((outgoing.get(id)?.length ?? 0) > 0 && row.end === date) {
          current.end = Math.min(current.end, (slot + 1) / slotCount);
        }
        result.set(id, current);
      }
    }
  }
  return result;
}

function isWeekend(date: string): boolean {
  const day = parseISODate(date).getUTCDay();
  return day === 0 || day === 6;
}

function shortWeekday(date: string): string {
  return new Intl.DateTimeFormat("en", {
    weekday: "short",
    timeZone: "UTC",
  })
    .format(parseISODate(date))
    .slice(0, 2);
}

function dayOfMonth(date: string): string {
  return String(parseISODate(date).getUTCDate());
}

function formatDisplayDate(value: string | undefined): string {
  if (!value) return "";
  const date = parseISODate(value);
  return `${date.getUTCDate()} ${shortMonthNames[date.getUTCMonth()]} ${date.getUTCFullYear()}`;
}

function monthSegments(
  days: string[],
): { key: string; label: string; dayCount: number }[] {
  const segments: { key: string; label: string; dayCount: number }[] = [];
  for (const date of days) {
    const key = date.slice(0, 7);
    const current = segments.at(-1);
    if (current?.key === key) {
      current.dayCount += 1;
      continue;
    }
    const parsed = parseISODate(date);
    segments.push({
      key,
      label: `${shortMonthNames[parsed.getUTCMonth()]} ${parsed.getUTCFullYear()}`,
      dayCount: 1,
    });
  }
  return segments;
}

function parseISODate(value: string): Date {
  return new Date(`${value}T00:00:00.000Z`);
}

function formatISODate(value: Date): string {
  return value.toISOString().slice(0, 10);
}

function addDays(value: string, amount: number): string {
  const date = parseISODate(value);
  date.setUTCDate(date.getUTCDate() + amount);
  return formatISODate(date);
}

function daysBetweenISO(from: string, to: string): number {
  return Math.round(
    (parseISODate(to).getTime() - parseISODate(from).getTime()) / 86_400_000,
  );
}

function formatEffort(minutes: number): string {
  return `${Math.round((minutes / 60) * 100) / 100}h`;
}

function sortFilters(filters: SavedPortfolioFilter[]): SavedPortfolioFilter[] {
  return [...filters].sort(
    (left, right) =>
      left.name.localeCompare(right.name, undefined, {
        sensitivity: "base",
      }) || left.id.localeCompare(right.id),
  );
}

function uniqueStrings(values: string[]): string[] {
  return [...new Set(values)];
}

function sameStrings(left: string[], right: string[]): boolean {
  return (
    left.length === right.length &&
    left.every((value, index) => value === right[index])
  );
}

function rowWindowForViewport(
  rowCount: number,
  scrollTop: number,
  clientHeight: number,
): { start: number; end: number } {
  const visibleCount = Math.max(1, Math.ceil(clientHeight / rowHeight));
  const start = Math.max(0, Math.floor(scrollTop / rowHeight) - rowOverscan);
  return {
    start,
    end: Math.min(
      rowCount,
      Math.max(
        start + maximumRenderedRows,
        start + visibleCount + rowOverscan * 2,
      ),
    ),
  };
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value));
}

function message(reason: unknown): string {
  return reason instanceof Error
    ? reason.message
    : "The saved filter could not be changed.";
}

function isAbort(reason: unknown): boolean {
  return reason instanceof DOMException && reason.name === "AbortError";
}
