import { useState } from "react";
import { useDateFormat } from "../hooks/useDateFormat";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Task } from "../types/task";
import { daysFromToday, toDateInputValue } from "../util/date";
import { KanbanCard } from "./KanbanCard";
import { SectionHeader } from "./SectionHeader";
import { TaskDialogs } from "./TaskDialogs";
import { UpcomingTaskDrawer } from "./UpcomingTaskDrawer";
import { ViewAllLink } from "./ViewAllLink";
import { FiCheckSquare } from "react-icons/fi";
import { Button } from "./Button";
import { EmptyState } from "./EmptyState";

type TaskListProps = {
    tasks: Task[];
    isLoading: boolean;
    isError: boolean;
    onCreate: () => void;
    // Shows only the first `limit` tasks, with a link to the tasks page.
    limit?: number;
    // Replaces the message shown when there are no tasks, e.g. for a filter.
    emptyMessage?: string;
    // Leaves out the section header, for a page that has its own title.
    hideHeader?: boolean;
    // One-line cards without the description, for a short preview.
    compact?: boolean;
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
    limit,
    emptyMessage,
    hideHeader = false,
    compact = false,
}: TaskListProps) {
    const t = useTranslate();
    const format = useDateFormat();
    const shown = limit === undefined ? tasks : tasks.slice(0, limit);
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
            {!hideHeader && (
                <SectionHeader
                    title={t("workspace.tasks.title")}
                    href="/tasks"
                    onCreate={onCreate}
                />
            )}
            {isLoading && (
                <p className="text-sm text-gray-500">{t("common.loading")}</p>
            )}
            {isError && (
                <p className="text-sm text-red-600">{t("common.error")}</p>
            )}
            {!isLoading && !isError && tasks.length === 0 && (
                <EmptyState
                    compact={compact || limit !== undefined}
                    icon={FiCheckSquare}
                    title={emptyMessage ?? t("workspace.tasks.empty")}
                    action={
                        limit === undefined &&
                        !emptyMessage && (
                            <Button
                                variant="primary"
                                size="lg"
                                onClick={onCreate}
                            >
                                + {t("workspace.tasks.create.title")}
                            </Button>
                        )
                    }
                />
            )}
            {shown.length > 0 && (
                <div className="flex flex-col gap-6">
                    {groupByDueDay(shown).map((group) => {
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
                                            <KanbanCard
                                                task={task}
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

            {shown.length < tasks.length && (
                <ViewAllLink href="/tasks" total={tasks.length} />
            )}

            {openTask && (
                <UpcomingTaskDrawer
                    task={openTask}
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
