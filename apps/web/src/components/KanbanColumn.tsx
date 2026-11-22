import { useState, type CSSProperties } from "react";
import {
    SortableContext,
    useSortable,
    verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { FiChevronsLeft, FiEdit2, FiPlus } from "react-icons/fi";
import { MdDragIndicator } from "react-icons/md";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { TaskGroup } from "../util/taskFilter";
import { dragIdOf } from "../util/taskView";
import type { PlanTask, TaskPropertyKey } from "../types/task";
import { KanbanCard } from "./KanbanCard";
import { QuickAddTask } from "./QuickAddTask";
import { Tooltip } from "./Tooltip";

type KanbanColumnProps = {
    group: TaskGroup;
    plain: boolean;
    statuses: Status[];
    projectId: string;
    planId: string;
    // Completed tasks of this status left out by the "hide completed" setting.
    hiddenCount: number;
    collapsed: boolean;
    visible: ReadonlySet<TaskPropertyKey>;
    onToggleCollapsed: () => void;
    onEdit: () => void;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
};

export function KanbanColumn({
    group,
    plain,
    statuses,
    projectId,
    planId,
    hiddenCount,
    collapsed,
    visible,
    onToggleCollapsed,
    onEdit,
    onTaskAction,
    onOpenTask,
}: KanbanColumnProps) {
    const t = useTranslate();
    const { status, tasks } = group;
    const title = group.label;
    const color = group.color ?? "var(--color-fg-placeholder)";
    const [isAdding, setIsAdding] = useState(false);
    // The column is both a sortable item (dragged by its handle) and the drop
    // target for tasks, so an empty column can still receive a task.
    const {
        attributes,
        listeners,
        setNodeRef,
        setActivatorNodeRef,
        transform,
        transition,
        isDragging,
        isOver,
        active,
    } = useSortable({
        // Only status columns can be dragged by their handle; the other
        // groups are plain drop targets.
        id: group.id,
        data: { type: status ? "status" : "group" },
        disabled: status ? false : { draggable: true },
    });
    const isTaskOver = isOver && active?.data.current?.type !== "status";
    const style = {
        transform: CSS.Translate.toString(transform),
        transition,
        "--status": color,
    } as CSSProperties;

    if (collapsed)
        return (
            <section
                ref={setNodeRef}
                aria-label={title}
                style={style}
                className="flex w-10 shrink-0 flex-col items-center rounded-lg bg-layer-1 py-2"
            >
                <button
                    type="button"
                    onClick={onToggleCollapsed}
                    aria-label={`${t("plan.group.expand")}: ${title}`}
                    aria-expanded={false}
                    className="flex cursor-pointer flex-col items-center gap-2 text-body-xs font-medium text-fg"
                >
                    <span
                        aria-hidden
                        className="size-2.5 rounded-full"
                        style={{ backgroundColor: color }}
                    />
                    <span className="[writing-mode:vertical-rl]">
                        {title}
                    </span>
                    <span className="text-fg-subtle">{tasks.length}</span>
                </button>
            </section>
        );

    return (
        <section
            ref={setNodeRef}
            aria-label={title}
            style={style}
            className={`flex max-h-full w-84 shrink-0 flex-col rounded-lg bg-layer-1 p-2 ${
                isTaskOver ? "ring-2 ring-accent/40" : ""
            } ${isDragging ? "relative z-10 opacity-80 shadow-overlay" : ""}`}
        >
            <header className="sticky top-0 z-10 flex h-9 shrink-0 items-center gap-2 px-1">
                {status && (
                <button
                    type="button"
                    ref={setActivatorNodeRef}
                    aria-label={`${t("plan.status.reorder")}: ${title}`}
                    title={t("plan.status.reorder")}
                    className={`-ml-1 touch-none rounded p-0.5 text-fg-placeholder hover:bg-layer-1-hover hover:text-fg ${
                        isDragging ? "cursor-grabbing" : "cursor-grab"
                    }`}
                    {...attributes}
                    {...listeners}
                >
                    <MdDragIndicator aria-hidden />
                </button>
                )}
                <span
                    aria-hidden
                    className="size-2.5 shrink-0 rounded-full"
                    style={{ backgroundColor: color }}
                />
                <h2 className="truncate text-body-xs font-medium text-fg">
                    {title}
                </h2>
                <span className="text-body-xs text-fg-subtle">
                    {tasks.length}
                </span>
                <div className="ml-auto flex items-center">
                    <Tooltip label={t("plan.group.collapse")}>
                        <button
                            type="button"
                            onClick={onToggleCollapsed}
                            aria-label={`${t("plan.group.collapse")}: ${title}`}
                            aria-expanded
                            className="cursor-pointer rounded p-1 text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                        >
                            <FiChevronsLeft aria-hidden />
                        </button>
                    </Tooltip>
                    {status && (
                    <Tooltip label={t("plan.task.add")}>
                        <button
                            type="button"
                            onClick={() => setIsAdding(true)}
                            aria-label={`${t("plan.task.add")}: ${title}`}
                            className="cursor-pointer rounded p-1 text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                        >
                            <FiPlus aria-hidden />
                        </button>
                    </Tooltip>
                    )}
                    {status && (
                    <Tooltip label={t("plan.status.edit.title")}>
                        <button
                            type="button"
                            onClick={onEdit}
                            aria-label={`${t("plan.status.edit.title")}: ${title}`}
                            className="cursor-pointer rounded p-1 text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                        >
                            <FiEdit2 aria-hidden />
                        </button>
                    </Tooltip>
                    )}
                </div>
            </header>
            <SortableContext
                id={group.id}
                items={tasks.map((task) => dragIdOf(plain, group.id, task.id))}
                strategy={plain ? () => null : verticalListSortingStrategy}
            >
                <ul className="flex min-h-12 flex-1 flex-col gap-2 overflow-y-auto py-1">
                    {tasks.map((task) => (
                        <SortableCard
                            key={task.id}
                            task={task}
                            dragId={dragIdOf(plain, group.id, task.id)}
                            groupId={group.id}
                            statuses={statuses}
                            projectId={projectId}
                            planId={planId}
                            visible={visible}
                            onAction={(action) => onTaskAction(task, action)}
                            onOpen={() => onOpenTask(task)}
                        />
                    ))}
                    {tasks.length === 0 && hiddenCount === 0 && (
                        <li className="flex h-12 items-center justify-center rounded-md border border-dashed border-line-strong text-caption text-fg-placeholder">
                            {t("plan.column.empty")}
                        </li>
                    )}
                    {hiddenCount > 0 && (
                        <li className="px-1 text-caption text-fg-placeholder">
                            {t("plan.column.hiddenDone").replace(
                                "{count}",
                                String(hiddenCount),
                            )}
                        </li>
                    )}
                </ul>
            </SortableContext>
            {status && (
                <QuickAddTask
                    projectId={projectId}
                    planId={planId}
                    status={status}
                    variant="kanban"
                    isOpen={isAdding}
                    onOpenChange={setIsAdding}
                />
            )}
        </section>
    );
}

function SortableCard({
    task,
    dragId,
    groupId,
    statuses,
    projectId,
    planId,
    visible,
    onAction,
    onOpen,
}: {
    task: PlanTask;
    dragId: string;
    groupId: string;
    statuses: Status[];
    projectId: string;
    planId: string;
    visible: ReadonlySet<TaskPropertyKey>;
    onAction: (action: CardAction) => void;
    onOpen: () => void;
}) {
    const isTemp = task.id.startsWith("temp-");
    const {
        attributes,
        listeners,
        setNodeRef,
        transform,
        transition,
        isDragging,
    } = useSortable({
        id: dragId,
        data: { type: "task", taskId: task.id, groupId },
        disabled: isTemp,
    });

    return (
        <li
            ref={setNodeRef}
            style={{ transform: CSS.Translate.toString(transform), transition }}
            className={`rounded-lg focus:outline-2 focus:outline-accent ${
                isDragging ? "opacity-40" : isTemp ? "opacity-60" : ""
            }`}
            {...attributes}
            {...listeners}
            // dnd-kit swallows the click that ends a drag, so this only
            // fires on a plain click.
            onClick={isTemp ? undefined : onOpen}
            onKeyDown={(event) => {
                listeners?.onKeyDown?.(event);
                if (event.key === "Enter" && event.target === event.currentTarget)
                    onOpen();
            }}
        >
            <KanbanCard
                task={task}
                statuses={statuses}
                projectId={projectId}
                planId={planId}
                visible={visible}
                onAction={isTemp ? undefined : onAction}
                isDraggable={!isTemp}
            />
        </li>
    );
}
