import { describe, expect, it } from "vitest";
import type { WBSNode } from "./wbs";
import { summarizeWBS } from "./wbsSummary";

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

function group(
  id: string,
  children: WBSNode[],
  executable: Partial<WBSNode["executable"]> = {},
): WBSNode {
  return {
    id,
    projectId: "project",
    name: id,
    position: 1,
    hasChildren: true,
    executable: {
      lagDays: 0,
      executionTimeline: {},
      commitmentTimeline: {},
      ...executable,
    },
    children,
  };
}

const direct = task("direct", {
  effortMinutes: 960,
  actualStart: "2026-08-03",
  actualEnd: "2026-08-03",
  executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
  commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
});
const nestedKnown = task("nested-known", {
  effortMinutes: 1440,
  executionTimeline: { start: "2026-08-04", end: "2026-08-06" },
  commitmentTimeline: { start: "2026-08-04", end: "2026-08-08" },
});
const nestedMissingCompleted = task("nested-missing-completed", {
  actualStart: "2026-08-10",
  actualEnd: "2026-08-10",
  commitmentTimeline: { start: "2026-08-09", end: "2026-08-12" },
});
const deepKnownCompleted = task("deep-known-completed", {
  effortMinutes: 480,
  actualStart: "2026-08-12",
  actualEnd: "2026-08-12",
  executionTimeline: { start: "2026-08-08", end: "2026-08-12" },
});
const missingUnfinished = task("missing-unfinished");

const realisticRoots = [
  group(
    "root-group",
    [
      direct,
      group("nested-group", [
        nestedKnown,
        group("deep-group", [nestedMissingCompleted, deepKnownCompleted]),
      ]),
      missingUnfinished,
    ],
    {
      effortMinutes: 9999,
      actualStart: "2026-01-01",
      actualEnd: "2026-01-01",
      executionTimeline: { start: "2020-01-01", end: "2030-01-01" },
      commitmentTimeline: { start: "2020-01-01", end: "2030-01-01" },
    },
  ),
];

describe("US-4.3 WBS summary aggregation", () => {
  it("AC-1..15 AC-28 AC-38 recursively aggregates Tasks once and ignores Group executable payload", () => {
    expect(summarizeWBS(realisticRoots)).toEqual({
      taskCount: 5,
      execution: {
        start: "2026-08-01",
        end: "2026-08-12",
        scheduledTaskCount: 3,
      },
      commitment: {
        start: "2026-08-01",
        end: "2026-08-12",
        scheduledTaskCount: 3,
      },
      completedKnownEffortMinutes: 1440,
      totalKnownEffortMinutes: 2880,
      taskWithoutEffortCount: 2,
      completionPercentage: 50,
    });
  });

  it("AC-3 keeps aggregates independent from child order and leaves source arrays untouched", () => {
    const roots = realisticRoots;
    const originalChildren = [...roots[0]!.children];
    const reordered = [
      group("root-group", [...originalChildren].reverse(), {
        effortMinutes: 9999,
      }),
    ];

    expect(summarizeWBS(reordered)).toEqual(summarizeWBS(roots));
    expect(roots[0]!.children).toEqual(originalChildren);
  });

  it("AC-4..10 ignores partial pairs and keeps Execution and Commitment ranges independent", () => {
    expect(
      summarizeWBS([
        task("execution-early", {
          executionTimeline: { start: "2026-08-01", end: "2026-08-04" },
          commitmentTimeline: { start: "2026-08-01" },
        }),
        task("execution-late", {
          executionTimeline: { start: "2026-08-03", end: "2026-08-10" },
          commitmentTimeline: { start: "2026-08-05", end: "2026-08-12" },
        }),
        task("commitment-early", {
          executionTimeline: { end: "2026-08-20" },
          commitmentTimeline: { start: "2026-08-02", end: "2026-08-08" },
        }),
        task("execution-only", {
          executionTimeline: { start: "2026-08-06", end: "2026-08-11" },
        }),
      ]),
    ).toMatchObject({
      taskCount: 4,
      execution: {
        start: "2026-08-01",
        end: "2026-08-11",
        scheduledTaskCount: 3,
      },
      commitment: {
        start: "2026-08-02",
        end: "2026-08-12",
        scheduledTaskCount: 2,
      },
    });
  });

  it("AC-6 AC-9 AC-16 AC-27 handles no schedules, all missing Effort, and a defensive empty Group", () => {
    const noKnownEffort = summarizeWBS([
      task("one"),
      task("two", { actualStart: "2026-08-01", actualEnd: "2026-08-01" }),
      task("three"),
    ]);
    expect(noKnownEffort).toEqual({
      taskCount: 3,
      execution: { scheduledTaskCount: 0 },
      commitment: { scheduledTaskCount: 0 },
      completedKnownEffortMinutes: 0,
      totalKnownEffortMinutes: 0,
      taskWithoutEffortCount: 3,
    });

    expect(summarizeWBS([group("empty", [])]).taskCount).toBe(0);
  });

  it("AC-13 preserves integer-minute precision for 0%, one decimal, half hours, and 100%", () => {
    expect(
      summarizeWBS([
        task("done", {
          effortMinutes: 60,
          actualStart: "2026-08-01",
          actualEnd: "2026-08-01",
        }),
        task("todo", { effortMinutes: 120 }),
      ]).completionPercentage,
    ).toBe(33.3);
    expect(summarizeWBS([task("half", { effortMinutes: 750 })])).toMatchObject({
      completedKnownEffortMinutes: 0,
      totalKnownEffortMinutes: 750,
      completionPercentage: 0,
    });
    expect(
      summarizeWBS([
        task("complete-half", {
          effortMinutes: 750,
          actualStart: "2026-08-01",
          actualEnd: "2026-08-01",
        }),
      ]).completionPercentage,
    ).toBe(100);
  });

  it("AC-18 removes reopened Effort only from the numerator", () => {
    const completed = task("reopened", {
      effortMinutes: 960,
      actualStart: "2026-08-01",
      actualEnd: "2026-08-01",
    });
    const other = task("other", { effortMinutes: 480 });
    expect(summarizeWBS([completed, other])).toMatchObject({
      completedKnownEffortMinutes: 960,
      totalKnownEffortMinutes: 1440,
      completionPercentage: 66.7,
    });

    const reopened = {
      ...completed,
      executable: {
        ...completed.executable,
        actualStart: undefined,
        actualEnd: undefined,
      },
    };
    expect(summarizeWBS([reopened, other])).toMatchObject({
      completedKnownEffortMinutes: 0,
      totalKnownEffortMinutes: 1440,
      completionPercentage: 0,
    });
  });
});
