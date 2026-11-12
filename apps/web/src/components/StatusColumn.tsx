import type { CSSProperties } from "react";
import {
    SortableContext,
    useSortable,
    verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { FiEdit2, FiFlag, FiPlus } from "react-icons/fi";
import { MdDragIndicator } from "react-icons/md";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import type { CardAction } from "../types/cardAction";
import type { PlanTask } from "../types/task";
import { SortableTaskCard } from "./SortableTaskCard";
import type { BoardView } from "./ViewToggle";

type StatusColumnProps = {
    status: Status;
    tasks: PlanTask[];
    layout: BoardView;
    onAddTask: () => void;
    onEdit: () => void;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
};

export function StatusColumn({
    status,
    tasks,
    layout,
    onAddTask,
    onEdit,
    onTaskAction,
    onOpenTask,
}: StatusColumnProps) {
    const t = useTranslate();
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
    } = useSortable({ id: status.id, data: { type: "status" } });
    const isBoard = layout === "board";
    const isTaskOver = isOver && active?.data.current?.type !== "status";

    return (
        <section
            ref={setNodeRef}
            aria-label={status.title}
            style={
                {
                    transform: CSS.Translate.toString(transform),
                    transition,
                    // Read by the tint classes below; any CSS color string
                    // the backend stores works with color-mix.
                    "--status": status.color,
                } as CSSProperties
            }
            // Tint the column by mixing the status color into the theme
            // surface (`--color-white` is the dark surface in dark mode), a
            // little stronger in dark mode so the tint stays visible.
            className={`flex flex-col rounded-lg border border-t-4 border-t-(--status) bg-[color-mix(in_srgb,var(--status)_8%,var(--color-white))] dark:bg-[color-mix(in_srgb,var(--status)_12%,var(--color-white))] ${
                isTaskOver
                    ? "border-[color-mix(in_srgb,var(--status)_70%,var(--color-white))]"
                    : "border-[color-mix(in_srgb,var(--status)_30%,var(--color-white))] dark:border-[color-mix(in_srgb,var(--status)_40%,var(--color-white))]"
            } ${
                isBoard ? "max-h-full w-72 shrink-0" : "w-full"
            } ${isDragging ? "relative z-10 opacity-80 shadow-lg" : ""}`}
        >
            <header className="flex items-center gap-2 px-3 pt-3 pb-2">
                <button
                    type="button"
                    ref={setActivatorNodeRef}
                    aria-label={`${t("plan.status.reorder")}: ${status.title}`}
                    title={t("plan.status.reorder")}
                    className={`-ml-1 touch-none rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-700 ${
                        isDragging ? "cursor-grabbing" : "cursor-grab"
                    }`}
                    {...attributes}
                    {...listeners}
                >
                    <MdDragIndicator aria-hidden />
                </button>
                <span
                    aria-hidden
                    className="h-2.5 w-2.5 shrink-0 rounded-full"
                    style={{ backgroundColor: status.color }}
                />
                <h2 className="truncate text-sm font-semibold">
                    {status.title}
                </h2>
                <span className="text-xs text-gray-500">{tasks.length}</span>
                {status.category === "done" && (
                    <span
                        title={t("plan.status.doneBadgeHint")}
                        className="flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs text-green-700"
                    >
                        <FiFlag aria-hidden />
                        {t("plan.status.doneBadge")}
                    </span>
                )}
                <button
                    type="button"
                    onClick={onEdit}
                    aria-label={`${t("plan.status.edit.title")}: ${status.title}`}
                    title={t("plan.status.edit.title")}
                    className="ml-auto cursor-pointer rounded p-1 text-gray-500 hover:bg-gray-200 hover:text-gray-900"
                >
                    <FiEdit2 aria-hidden />
                </button>
                <button
                    type="button"
                    onClick={onAddTask}
                    aria-label={`${t("plan.task.add")}: ${status.title}`}
                    title={t("plan.task.add")}
                    className="cursor-pointer rounded p-1 text-gray-500 hover:bg-gray-200 hover:text-gray-900"
                >
                    <FiPlus aria-hidden />
                </button>
            </header>
            {status.description && (
                <p
                    title={status.description}
                    className="line-clamp-2 px-3 pb-2 text-xs break-words text-gray-600"
                >
                    {status.description}
                </p>
            )}
            <SortableContext
                id={status.id}
                items={tasks.map((task) => task.id)}
                strategy={verticalListSortingStrategy}
            >
                <ul
                    className={`flex min-h-16 flex-col gap-2 px-3 pb-3 ${isBoard ? "overflow-y-auto" : ""}`}
                >
                    {tasks.map((task) => (
                        <SortableTaskCard
                            key={task.id}
                            task={task}
                            layout={layout}
                            onAction={(action) => onTaskAction(task, action)}
                            onOpen={() => onOpenTask(task)}
                        />
                    ))}
                    {tasks.length === 0 && (
                        <li className="flex h-16 items-center justify-center rounded-md border border-dashed border-gray-300 text-xs text-gray-400">
                            {t("plan.column.empty")}
                        </li>
                    )}
                </ul>
            </SortableContext>
            {isBoard && (
                <button
                    type="button"
                    onClick={onAddTask}
                    className="mx-3 mb-3 flex cursor-pointer items-center gap-1 rounded-md px-2 py-1.5 text-sm text-gray-500 hover:bg-gray-200 hover:text-gray-900"
                >
                    <FiPlus aria-hidden />
                    {t("plan.task.add")}
                </button>
            )}
        </section>
    );
}
