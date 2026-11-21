import { useState } from "react";
import { FiChevronDown, FiChevronRight, FiPlus } from "react-icons/fi";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { useDroppable } from "@dnd-kit/core";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask, TaskPropertyKey } from "../types/task";
import { ListRow } from "./ListRow";
import { QuickAddTask } from "./QuickAddTask";

type ListGroupProps = {
    status: Status;
    tasks: PlanTask[];
    statuses: Status[];
    projectId: string;
    planId: string;
    hiddenCount: number;
    collapsed: boolean;
    visible: ReadonlySet<TaskPropertyKey>;
    activeTaskId: string | null;
    onToggleCollapsed: () => void;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
};

export function ListGroup({
    status,
    tasks,
    statuses,
    projectId,
    planId,
    hiddenCount,
    collapsed,
    visible,
    activeTaskId,
    onToggleCollapsed,
    onTaskAction,
    onOpenTask,
}: ListGroupProps) {
    const t = useTranslate();
    const [isAdding, setIsAdding] = useState(false);
    // The group is the drop target for its tasks, so an empty or collapsed
    // group can still receive one.
    const { setNodeRef, isOver } = useDroppable({ id: status.id });

    return (
        <section
            ref={setNodeRef}
            aria-label={status.title}
            className={isOver ? "bg-accent/5" : ""}
        >
            <header className="sticky top-0 z-10 flex h-9 items-center gap-2 border-b border-line bg-layer-1 px-4">
                <button
                    type="button"
                    onClick={onToggleCollapsed}
                    aria-expanded={!collapsed}
                    aria-label={`${t(collapsed ? "plan.group.expand" : "plan.group.collapse")}: ${status.title}`}
                    className="cursor-pointer text-fg-subtle hover:text-fg"
                >
                    {collapsed ? (
                        <FiChevronRight aria-hidden />
                    ) : (
                        <FiChevronDown aria-hidden />
                    )}
                </button>
                <span
                    aria-hidden
                    className="size-2.5 shrink-0 rounded-full"
                    style={{ backgroundColor: status.color }}
                />
                <h2 className="text-body-xs font-medium text-fg">
                    {status.title}
                </h2>
                <span className="text-body-xs text-fg-subtle">
                    {tasks.length}
                </span>
                <button
                    type="button"
                    onClick={() => {
                        if (collapsed) onToggleCollapsed();
                        setIsAdding(true);
                    }}
                    aria-label={`${t("plan.task.add")}: ${status.title}`}
                    title={t("plan.task.add")}
                    className="ml-auto cursor-pointer rounded p-1 text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                >
                    <FiPlus aria-hidden />
                </button>
            </header>
            {!collapsed && (
                <>
                    <SortableContext
                        id={status.id}
                        items={tasks.map((task) => task.id)}
                        strategy={verticalListSortingStrategy}
                    >
                        <ul>
                            {tasks.map((task) => (
                                <ListRow
                                    key={task.id}
                                    task={task}
                                    projectId={projectId}
                                    planId={planId}
                                    statuses={statuses}
                                    visible={visible}
                                    isActive={task.id === activeTaskId}
                                    onAction={(action) => onTaskAction(task, action)}
                                    onOpen={() => onOpenTask(task)}
                                />
                            ))}
                        </ul>
                    </SortableContext>
                    {hiddenCount > 0 && (
                        <p className="px-4 py-2 text-caption text-fg-placeholder">
                            {t("plan.column.hiddenDone").replace(
                                "{count}",
                                String(hiddenCount),
                            )}
                        </p>
                    )}
                    <QuickAddTask
                        projectId={projectId}
                        planId={planId}
                        status={status}
                        variant="list"
                        isOpen={isAdding}
                        onOpenChange={setIsAdding}
                    />
                </>
            )}
        </section>
    );
}
