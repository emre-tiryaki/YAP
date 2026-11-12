import { usePlanStatuses } from "../hooks/usePlans";
import { usePlanTasks } from "../hooks/useTasks";
import type { Task } from "../types/task";
import { TaskDetailModal } from "./TaskDetailModal";

type UpcomingTaskDrawerProps = {
    task: Task;
    onDelete: () => void;
    onClose: () => void;
};

// Opens the task detail dialog for a task shown outside its plan page. The
// dialog needs the plan's statuses and tasks, so they are loaded here.
export function UpcomingTaskDrawer({
    task,
    onDelete,
    onClose,
}: UpcomingTaskDrawerProps) {
    const statuses = usePlanStatuses(task.projectId, task.planId);
    const tasks = usePlanTasks(task.projectId, task.planId);
    // Read from the plan query so the dialog shows fresh values and stays
    // open after the task leaves the upcoming list (e.g. marked done).
    const planTask = tasks.data?.find((other) => other.id === task.id);

    if (!statuses.data || !tasks.data || !planTask) return null;

    return (
        <TaskDetailModal
            projectId={task.projectId}
            planId={task.planId}
            task={planTask}
            statuses={statuses.data}
            tasks={tasks.data}
            onDelete={onDelete}
            onClose={onClose}
        />
    );
}
