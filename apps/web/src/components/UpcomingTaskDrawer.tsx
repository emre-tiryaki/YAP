import type { Task } from "../types/task";
import { TaskPeek } from "./TaskPeek";

type UpcomingTaskDrawerProps = {
    task: Task;
    onClose: () => void;
};

// Opens a task shown outside its plan page (the upcoming list) in the peek.
export function UpcomingTaskDrawer({ task, onClose }: UpcomingTaskDrawerProps) {
    return (
        <TaskPeek
            projectId={task.projectId}
            planId={task.planId}
            taskId={task.id}
            onClose={onClose}
        />
    );
}
