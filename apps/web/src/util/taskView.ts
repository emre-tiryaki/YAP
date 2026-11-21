import type { TaskPropertyKey } from "../types/task";

// How completed tasks are shown inside a status (see the "Categories"
// settings). Plain functions so the board and the list share them.

export const allTaskProperties: ReadonlySet<TaskPropertyKey> = new Set([
    "status",
    "priority",
    "dueDate",
    "subtasks",
    "comments",
]);

export type TaskViewOptions = {
    hideDone: boolean;
    doneToBottom: boolean;
};

type ViewTask = { id: string; completedAt: Date | null };

function isDone(task: ViewTask): boolean {
    return task.completedAt !== null;
}

// The tasks of one status in display order: completed ones left out or moved
// after the open ones (keeping their order), depending on the options.
export function arrangeTasks<T extends ViewTask>(
    tasks: T[],
    options: TaskViewOptions,
): T[] {
    if (options.hideDone) return tasks.filter((task) => !isDone(task));
    if (options.doneToBottom)
        return [
            ...tasks.filter((task) => !isDone(task)),
            ...tasks.filter(isDone),
        ];
    return tasks;
}

// Turns a drop position in the displayed column into the index the backend
// expects: the position among all other tasks of the status, hidden ones
// included, in stored order.
//
// The moved task is anchored to a displayed neighbour: after the previous
// task when that one is in the same group (open or completed) as the moved
// task, otherwise before the next one, otherwise after the previous one.
export function storedIndex(
    displayed: string[],
    taskId: string,
    moved: ViewTask,
    siblings: ViewTask[],
): number {
    const ids = siblings.map((task) => task.id);
    const at = displayed.indexOf(taskId);
    const prevId = at > 0 ? displayed[at - 1] : undefined;
    const nextId = at >= 0 ? displayed[at + 1] : undefined;
    const prev = siblings.find((task) => task.id === prevId);
    const next = siblings.find((task) => task.id === nextId);

    if (prev && isDone(prev) === isDone(moved)) return ids.indexOf(prev.id) + 1;
    if (next) return ids.indexOf(next.id);
    if (prev) return ids.indexOf(prev.id) + 1;
    return ids.length;
}
