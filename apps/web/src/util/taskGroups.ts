import type { Status } from "../types/status";
import type { PlanTask } from "../types/task";
import { arrangeTasks, type TaskViewOptions } from "./taskView";

export type TaskGroup = {
    id: string;
    label: string;
    color: string;
    status: Status;
    // In display order: the server's order with the completed-task settings
    // applied.
    tasks: PlanTask[];
};

// Phase 1 groups by status only. A group per status, in the given order, even
// when it has no tasks.
export function groupTasks(
    tasks: PlanTask[],
    statuses: Status[],
    options: TaskViewOptions,
): TaskGroup[] {
    return statuses.map((status) => ({
        id: status.id,
        label: status.title,
        color: status.color,
        status,
        tasks: arrangeTasks(
            tasks.filter((task) => task.statusId === status.id),
            options,
        ),
    }));
}
