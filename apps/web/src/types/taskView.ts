import type { TaskPriority, TaskPropertyKey } from "./task";

export type DueFilter = "overdue" | "today" | "week" | "none";
export type DoneFilter = "open" | "done" | "all";

export type TaskFilters = {
    // Status ids; empty means all.
    status: string[];
    priority: TaskPriority[];
    // Label ids.
    labels: string[];
    due: DueFilter | null;
    done: DoneFilter;
};

export type GroupBy = "status" | "priority" | "label" | "due" | "none";
export type OrderBy =
    | "manual"
    | "created"
    | "updated"
    | "due"
    | "priority"
    | "title";
export type OrderDir = "asc" | "desc";

export type DisplayOptions = {
    groupBy: GroupBy;
    orderBy: OrderBy;
    orderDir: OrderDir;
    properties: TaskPropertyKey[];
    showEmptyGroups: boolean;
};
