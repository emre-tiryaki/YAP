import { useChangeTaskStatus } from "../hooks/useChangeTaskStatus";
import { useUpdateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import type { PlanTask } from "../types/task";
import { daysFromToday, fromDateInputValue, toDateInputValue } from "../util/date";
import { DatePicker } from "./DatePicker";
import { Pill } from "./Pill";
import { PrioritySelect } from "./PrioritySelect";
import { PropertySelect } from "./PropertySelect";

type TaskMetaRowProps = {
    task: PlanTask;
    projectId: string;
    planId: string;
    statuses: Status[];
    dueDateRequest: number;
};

function Divider() {
    return <span className="h-4 w-px bg-line" aria-hidden />;
}

// The compact property controls under the title in the peek.
export function TaskMetaRow({ task, projectId, planId, statuses, dueDateRequest }: TaskMetaRowProps) {
    const t = useTranslate();
    const update = useUpdateTask();
    const changeStatus = useChangeTaskStatus(projectId, planId, statuses);
    const isDone = task.completedAt !== null;
    const isOverdue =
        task.dueDate !== null && !isDone && daysFromToday(task.dueDate) < 0;

    return (
        <div className="flex flex-wrap items-center gap-1">
            <PropertySelect
                variant="ghost"
                ariaLabel={t("plan.task.create.status")}
                placeholder={t("plan.task.create.status")}
                value={task.statusId}
                onChange={(value) =>
                    typeof value === "string" &&
                    !changeStatus.isPending &&
                    changeStatus.change(task, value)
                }
                options={statuses.map((status) => ({
                    value: status.id,
                    label: status.title,
                    icon: (
                        <span
                            aria-hidden
                            className="size-2.5 rounded-full"
                            style={{ backgroundColor: status.color }}
                        />
                    ),
                }))}
            />
            <Divider />
            <PrioritySelect
                variant="ghost"
                value={task.priority}
                onChange={(priority) =>
                    update.mutate({ taskId: task.id, input: { priority } })
                }
            />
            <Divider />
            <DatePicker
                label={t("form.dueDate")}
                value={toDateInputValue(task.dueDate)}
                pill={{ overdue: isOverdue }}
                openRequest={dueDateRequest}
                onChange={(value) =>
                    update.mutate({
                        taskId: task.id,
                        input: { dueDate: fromDateInputValue(value) },
                    })
                }
            />
            <Divider />
            <Pill>{t(`status.category.${task.statusCategory}`)}</Pill>
        </div>
    );
}
