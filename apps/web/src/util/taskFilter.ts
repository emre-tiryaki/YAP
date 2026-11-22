import type { Status } from "../types/status";
import type { Label } from "../types/label";
import type { PlanTask, TaskPriority } from "../types/task";
import type {
    DisplayOptions,
    OrderBy,
    OrderDir,
    TaskFilters,
} from "../types/taskView";
import { daysFromToday } from "./date";
import { priorities } from "./priority";

export const emptyFilters: TaskFilters = {
    status: [],
    priority: [],
    labels: [],
    due: null,
    done: "all",
};

export function hasActiveFilters(filters: TaskFilters): boolean {
    return (
        filters.status.length > 0 ||
        filters.priority.length > 0 ||
        filters.labels.length > 0 ||
        filters.due !== null ||
        filters.done !== "all"
    );
}

export function filterTasks<T extends PlanTask>(
    tasks: T[],
    filters: TaskFilters,
    now: Date = new Date(),
): T[] {
    return tasks.filter((task) => {
        if (filters.status.length > 0 && !filters.status.includes(task.statusId))
            return false;
        if (
            filters.priority.length > 0 &&
            !filters.priority.includes(task.priority)
        )
            return false;
        if (
            filters.labels.length > 0 &&
            !task.labelIds.some((id) => filters.labels.includes(id))
        )
            return false;
        if (filters.done === "open" && task.completedAt !== null) return false;
        if (filters.done === "done" && task.completedAt === null) return false;
        if (filters.due !== null) {
            if (filters.due === "none") return task.dueDate === null;
            if (task.dueDate === null) return false;
            const days = daysFromToday(task.dueDate, now);
            if (filters.due === "overdue")
                return days < 0 && task.completedAt === null;
            if (filters.due === "today") return days === 0;
            return days >= 0 && days <= 6;
        }
        return true;
    });
}

const priorityRank = new Map<TaskPriority, number>(
    priorities.map((p, index) => [p.value, index]),
);

// Compares two tasks on one key, ascending. Tasks without a due date sort
// after dated ones in both directions (handled by the caller).
function compare(a: PlanTask, b: PlanTask, orderBy: OrderBy): number {
    switch (orderBy) {
        case "created":
            return a.createdAt.getTime() - b.createdAt.getTime();
        case "updated":
            return a.updatedAt.getTime() - b.updatedAt.getTime();
        case "priority":
            return (
                (priorityRank.get(a.priority) ?? 0) -
                (priorityRank.get(b.priority) ?? 0)
            );
        case "title":
            return a.title.localeCompare(b.title, undefined, {
                sensitivity: "base",
                numeric: true,
            });
        default:
            return 0;
    }
}

// "manual" keeps the given (stored) order. Sorting is stable, so ties keep
// that order too.
export function orderTasks(
    tasks: PlanTask[],
    orderBy: OrderBy,
    dir: OrderDir,
): PlanTask[] {
    if (orderBy === "manual") return tasks;
    const sign = dir === "asc" ? 1 : -1;
    return [...tasks].sort((a, b) => {
        if (orderBy === "due") {
            if (a.dueDate === null && b.dueDate === null) return 0;
            if (a.dueDate === null) return 1;
            if (b.dueDate === null) return -1;
            return sign * (a.dueDate.getTime() - b.dueDate.getTime());
        }
        // Priority "ascending" reads urgent first, so the sign is natural.
        return sign * compare(a, b, orderBy);
    });
}

export type TaskGroup = {
    // Unique among the groups of one view, and never equal to a task id.
    id: string;
    label: string;
    // Dot colour (any CSS colour) or null for none.
    color: string | null;
    // Set when the group is a status, so a task can be added to it.
    status: Status | null;
    tasks: PlanTask[];
};

export type GroupContext = {
    statuses: Status[];
    labels: Label[];
    // Display names of the fixed groups; looked up through t() by the caller.
    names: {
        all: string;
        noLabel: string;
        priority: Record<TaskPriority, string>;
        due: Record<"overdue" | "today" | "week" | "later" | "none", string>;
    };
};

const dueColors = {
    overdue: "#dc2626",
    today: "#f59e0b",
    week: "#3b82f6",
    later: "#6b7280",
    none: null,
} as const;

const labelColorValues: Record<string, string> = {
    gray: "#6b7280",
    red: "#ef4444",
    orange: "#f97316",
    amber: "#f59e0b",
    emerald: "#10b981",
    teal: "#14b8a6",
    blue: "#3b82f6",
    indigo: "#6366f1",
    violet: "#8b5cf6",
    pink: "#ec4899",
};

const priorityColors: Record<TaskPriority, string | null> = {
    urgent: "#dc2626",
    high: "#ea580c",
    medium: "#d97706",
    low: "#2563eb",
    none: null,
};

function dueBucket(
    task: PlanTask,
    now: Date,
): "overdue" | "today" | "week" | "later" | "none" {
    if (task.dueDate === null) return "none";
    const days = daysFromToday(task.dueDate, now);
    if (days < 0) return task.completedAt === null ? "overdue" : "later";
    if (days === 0) return "today";
    return days <= 6 ? "week" : "later";
}

// Splits the tasks into groups. The tasks keep their relative order inside a
// group. Empty groups are left out unless `showEmptyGroups` is set.
export function groupTasks(
    tasks: PlanTask[],
    display: Pick<DisplayOptions, "groupBy" | "showEmptyGroups">,
    context: GroupContext,
    now: Date = new Date(),
): TaskGroup[] {
    let groups: TaskGroup[];
    switch (display.groupBy) {
        case "status":
            groups = context.statuses.map((status) => ({
                id: status.id,
                label: status.title,
                color: status.color,
                status,
                tasks: tasks.filter((task) => task.statusId === status.id),
            }));
            break;
        case "priority":
            groups = priorities.map((p) => ({
                id: `priority:${p.value}`,
                label: context.names.priority[p.value],
                color: priorityColors[p.value],
                status: null,
                tasks: tasks.filter((task) => task.priority === p.value),
            }));
            break;
        case "label":
            // A task sits in the group of each of its labels.
            groups = [
                ...context.labels.map((label) => ({
                    id: `label:${label.id}`,
                    label: label.name,
                    color: labelColorValues[label.color] ?? null,
                    status: null,
                    tasks: tasks.filter((task) =>
                        task.labelIds.includes(label.id),
                    ),
                })),
                {
                    id: "label:none",
                    label: context.names.noLabel,
                    color: null,
                    status: null,
                    tasks: tasks.filter((task) => task.labelIds.length === 0),
                },
            ];
            break;
        case "due":
            groups = (
                ["overdue", "today", "week", "later", "none"] as const
            ).map((bucket) => ({
                id: `due:${bucket}`,
                label: context.names.due[bucket],
                color: dueColors[bucket],
                status: null,
                tasks: tasks.filter((task) => dueBucket(task, now) === bucket),
            }));
            break;
        default:
            groups = [
                {
                    id: "all",
                    label: context.names.all,
                    color: null,
                    status: null,
                    tasks,
                },
            ];
    }
    if (display.showEmptyGroups || display.groupBy === "none") return groups;
    return groups.filter((group) => group.tasks.length > 0);
}
