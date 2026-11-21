import type { Status } from "../types/status";
import type { PlanTask } from "../types/task";
import { useMoveTask, usePlanTasks } from "./useTasks";

// Moves a task to another status, at the end of it.
export function useChangeTaskStatus(
    projectId: string,
    planId: string,
    statuses: Status[],
) {
    const move = useMoveTask(projectId, planId);
    const tasks = usePlanTasks(projectId, planId);

    return {
        isPending: move.isPending,
        error: move.error,
        change(task: PlanTask, statusId: string) {
            const target = statuses.find((status) => status.id === statusId);
            if (!target || statusId === task.statusId) return;
            const index = (tasks.data ?? []).filter(
                (other) => other.statusId === statusId && other.id !== task.id,
            ).length;
            move.mutate({
                taskId: task.id,
                statusId,
                statusCategory: target.category,
                index,
            });
        },
    };
}
