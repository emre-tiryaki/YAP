import { useState } from "react";
import { useDateFormat } from "../hooks/useDateFormat";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Task } from "../types/task";
import { daysFromToday, toDateInputValue } from "../util/date";
import { PlanTaskCard } from "./PlanTaskCard";
import { SectionHeader } from "./SectionHeader";
import { TaskDialogs } from "./TaskDialogs";
import { UpcomingTaskDrawer } from "./UpcomingTaskDrawer";

type TaskListProps = {
    tasks: Task[];
    isLoading: boolean;
    isError: boolean;
    onCreate: () => void;
};

type TaskGroup = {
    key: string;
    // Null for the tasks without a due date.
    date: Date | null;
    tasks: Task[];
};

// Groups tasks by their local due day. The server sends dated tasks sorted
// by due date with the undated ones last, so groups come out in order.
function groupByDueDay(tasks: Task[]): TaskGroup[] {
    const groups: TaskGroup[] = [];
    for (const task of tasks) {
        const key = task.dueDate ? toDateInputValue(task.dueDate) : "none";
        const last = groups.at(-1);
        if (last?.key === key) last.tasks.push(task);
        else groups.push({ key, date: task.dueDate, tasks: [task] });
    }
    return groups;
}

export function TaskList({
    tasks,
    isError,
    isLoading,
    onCreate,
}: TaskListProps) {
    const t = useTranslate();
    const format = useDateFormat();
    const [openTask, setOpenTask] = useState<Task | null>(null);
    const [taskAction, setTaskAction] = useState<{
        task: Task;
        action: CardAction;
    } | null>(null);

    function groupLabel(date: Date | null) {
        if (!date) return t("workspace.tasks.noDueDate");
        const days = daysFromToday(date);
        if (days === 0) return t("workspace.tasks.today");
        if (days === 1) return t("workspace.tasks.tomorrow");
        return date.toLocaleDateString(format.locale, {
            weekday: "long",
            day: "numeric",
            month: "long",
        });
    }

    return (
        <section>
            <SectionHeader
                title={t("workspace.tasks.title")}
                onCreate={onCreate}
            />
            {isLoading && (
                <p className="text-sm text-gray-500">{t("common.loading")}</p>
            )}
            {isError && (
                <p className="text-sm text-red-600">{t("common.error")}</p>
            )}
            {!isLoading && !isError && tasks.length === 0 && (
                <p className="text-sm text-gray-500">
                    {t("workspace.tasks.empty")}
                </p>
            )}
            {tasks.length > 0 && (
                <div className="flex flex-col gap-6">
                    {groupByDueDay(tasks).map((group) => {
                        const isOverdue =
                            group.date !== null && daysFromToday(group.date) < 0;
                        return (
                            <div key={group.key}>
                                <h3
                                    className={`mb-2 flex items-center gap-2 text-xs font-semibold tracking-wide uppercase ${
                                        isOverdue
                                            ? "text-red-600"
                                            : "text-gray-500"
                                    }`}
                                >
                                    {groupLabel(group.date)}
                                    {isOverdue && (
                                        <span className="rounded bg-red-50 px-1.5 py-0.5 font-medium normal-case">
                                            {t("workspace.tasks.overdue")}
                                        </span>
                                    )}
                                    <span className="font-normal text-gray-400">
                                        {group.tasks.length}
                                    </span>
                                </h3>
                                <ul className="flex flex-col gap-2">
                                    {group.tasks.map((task) => (
                                        <li
                                            key={task.id}
                                            onClick={() => setOpenTask(task)}
                                        >
                                            <PlanTaskCard
                                                task={task}
                                                layout="board"
                                                isDraggable={false}
                                                projectId={task.projectId}
                                                planId={task.planId}
                                                onAction={(action) =>
                                                    setTaskAction({
                                                        task,
                                                        action,
                                                    })
                                                }
                                            />
                                        </li>
                                    ))}
                                </ul>
                            </div>
                        );
                    })}
                </div>
            )}

            {openTask && (
                <UpcomingTaskDrawer
                    task={openTask}
                    onDelete={() => {
                        setOpenTask(null);
                        setTaskAction({ task: openTask, action: "delete" });
                    }}
                    onClose={() => setOpenTask(null)}
                />
            )}
            {taskAction && (
                <TaskDialogs
                    task={taskAction.task}
                    action={taskAction.action}
                    onClose={() => setTaskAction(null)}
                />
            )}
        </section>
    );
}
