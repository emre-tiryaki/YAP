import { FiCheckSquare, FiMessageSquare } from "react-icons/fi";
import { TbLock } from "react-icons/tb";
import { useChangeTaskStatus } from "../hooks/useChangeTaskStatus";
import { useUpdateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import type { PlanTask, TaskPropertyKey } from "../types/task";
import { daysFromToday, fromDateInputValue, toDateInputValue } from "../util/date";
import { DatePicker } from "./DatePicker";
import { LabelsSelect } from "./LabelsSelect";
import { Pill } from "./Pill";
import { PrioritySelect } from "./PrioritySelect";
import { PropertySelect } from "./PropertySelect";

type TaskPropertiesProps = {
    task: PlanTask;
    projectId: string;
    planId: string;
    // Without the plan's statuses (e.g. in the upcoming list) the status pill
    // is left out.
    statuses?: Status[];
    visible: ReadonlySet<TaskPropertyKey>;
};

function StatusDot({ color }: { color: string }) {
    return (
        <span
            aria-hidden
            className="size-2.5 shrink-0 rounded-full"
            style={{ backgroundColor: color }}
        />
    );
}

// The inline properties of a task row or card. Each editable pill saves on its
// own and never opens the task.
export function TaskProperties({
    task,
    projectId,
    planId,
    statuses,
    visible,
}: TaskPropertiesProps) {
    const t = useTranslate();
    const update = useUpdateTask();
    const isDone = task.completedAt !== null;
    // A task that is still being created cannot be edited yet.
    const isTemp = task.id.startsWith("temp-");
    const hasSubtasks = task.subtaskCount > 0;
    const isOverdue =
        task.dueDate !== null && !isDone && daysFromToday(task.dueDate) < 0;

    return (
        <div className="flex flex-wrap items-center gap-1">
            {task.blocked && !isDone && (
                <Pill icon={<TbLock aria-hidden />} color="text-red-600">
                    {t("task.blocked")}
                </Pill>
            )}
            {visible.has("status") && statuses && !isTemp && (
                <StatusPill
                    task={task}
                    projectId={projectId}
                    planId={planId}
                    statuses={statuses}
                />
            )}
            {visible.has("priority") && !isTemp && (
                <PrioritySelect
                    variant="pill"
                    iconOnly={task.priority === "none"}
                    value={task.priority}
                    onChange={(priority) =>
                        update.mutate({ taskId: task.id, input: { priority } })
                    }
                />
            )}
            {visible.has("labels") && !isTemp && (
                <LabelsSelect
                    variant="pill"
                    projectId={projectId}
                    value={task.labelIds}
                    onChange={(labelIds) =>
                        update.mutate({ taskId: task.id, input: { labelIds } })
                    }
                />
            )}
            {visible.has("startDate") && task.startDate !== null && !isTemp && (
                <DatePicker
                    label={t("form.startDate")}
                    value={toDateInputValue(task.startDate)}
                    pill={{ overdue: false }}
                    onChange={(value) =>
                        update.mutate({
                            taskId: task.id,
                            input: { startDate: fromDateInputValue(value) },
                        })
                    }
                />
            )}
            {visible.has("dueDate") && !isTemp && (
                <DatePicker
                    label={t("form.dueDate")}
                    value={toDateInputValue(task.dueDate)}
                    pill={{ overdue: isOverdue }}
                    onChange={(value) =>
                        update.mutate({
                            taskId: task.id,
                            input: { dueDate: fromDateInputValue(value) },
                        })
                    }
                />
            )}
            {visible.has("subtasks") && hasSubtasks && (
                <Pill icon={<FiCheckSquare aria-hidden />}>
                    <span
                        title={t("task.card.subtasks")
                            .replace("{done}", String(task.doneSubtaskCount))
                            .replace("{total}", String(task.subtaskCount))}
                    >
                        {task.doneSubtaskCount}/{task.subtaskCount}
                    </span>
                </Pill>
            )}
            {visible.has("comments") && task.commentCount > 0 && (
                <Pill icon={<FiMessageSquare aria-hidden />}>
                    <span
                        title={t("task.card.comments").replace(
                            "{count}",
                            String(task.commentCount),
                        )}
                    >
                        {task.commentCount}
                    </span>
                </Pill>
            )}
        </div>
    );
}

function StatusPill({
    task,
    projectId,
    planId,
    statuses,
}: {
    task: PlanTask;
    projectId: string;
    planId: string;
    statuses: Status[];
}) {
    const t = useTranslate();
    const changeStatus = useChangeTaskStatus(projectId, planId, statuses);
    const current = statuses.find((status) => status.id === task.statusId);

    return (
        <PropertySelect
            variant="pill"
            ariaLabel={t("task.property.status")}
            tooltip={t("task.property.status")}
            placeholder={t("task.property.status")}
            value={task.statusId}
            triggerLabel={current?.title}
            onChange={(value) =>
                typeof value === "string" && changeStatus.change(task, value)
            }
            options={statuses.map((status) => ({
                value: status.id,
                label: status.title,
                icon: <StatusDot color={status.color} />,
            }))}
        />
    );
}
