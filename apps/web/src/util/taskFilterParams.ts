import type { TaskPriority } from "../types/task";
import type { DoneFilter, DueFilter, TaskFilters } from "../types/taskView";
import { emptyFilters } from "./taskFilter";

const priorityValues: readonly TaskPriority[] = [
    "urgent",
    "high",
    "medium",
    "low",
    "none",
];
const dueValues: readonly DueFilter[] = ["overdue", "today", "week", "none"];
const doneValues: readonly DoneFilter[] = ["open", "done", "all"];

function list(params: URLSearchParams, key: string): string[] {
    const raw = params.get(key);
    return raw ? raw.split(",").filter((item) => item !== "") : [];
}

// Reads the filters from the URL's query keys status, priority, label (comma
// lists), due and done. Unknown values are dropped.
export function filtersFromParams(params: URLSearchParams): TaskFilters {
    const due = params.get("due");
    const done = params.get("done");
    return {
        status: list(params, "status"),
        priority: list(params, "priority").filter((p): p is TaskPriority =>
            priorityValues.includes(p as TaskPriority),
        ),
        labels: list(params, "label"),
        due: dueValues.includes(due as DueFilter) ? (due as DueFilter) : null,
        done: doneValues.includes(done as DoneFilter)
            ? (done as DoneFilter)
            : emptyFilters.done,
    };
}

// Writes the filters into a copy of `params`, leaving other keys (view, task,
// project...) alone. Empty filters remove their key.
export function paramsWithFilters(
    params: URLSearchParams,
    filters: TaskFilters,
): URLSearchParams {
    const next = new URLSearchParams(params);
    function set(key: string, value: string) {
        if (value === "") next.delete(key);
        else next.set(key, value);
    }
    set("status", filters.status.join(","));
    set("priority", filters.priority.join(","));
    set("label", filters.labels.join(","));
    set("due", filters.due ?? "");
    set("done", filters.done === "all" ? "" : filters.done);
    return next;
}
