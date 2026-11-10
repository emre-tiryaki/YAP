import { useDroppable } from "@dnd-kit/core";
import {
    SortableContext,
    verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { FiFlag, FiPlus } from "react-icons/fi";
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
    onTaskAction: (task: PlanTask, action: CardAction) => void;
};

export function StatusColumn({
    status,
    tasks,
    layout,
    onAddTask,
    onTaskAction,
}: StatusColumnProps) {
    const t = useTranslate();
    const { setNodeRef, isOver } = useDroppable({ id: status.id });
    const isBoard = layout === "board";

    return (
        <section
            aria-label={status.title}
            className={`flex flex-col rounded-lg border bg-gray-50 ${
                isBoard ? "max-h-full w-72 shrink-0" : "w-full"
            } ${isOver ? "border-gray-400" : "border-gray-200"}`}
        >
            <header className="flex items-center gap-2 px-3 pt-3 pb-2">
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
                    onClick={onAddTask}
                    aria-label={`${t("plan.task.add")}: ${status.title}`}
                    title={t("plan.task.add")}
                    className="ml-auto cursor-pointer rounded p-1 text-gray-500 hover:bg-gray-200 hover:text-gray-900"
                >
                    <FiPlus aria-hidden />
                </button>
            </header>
            <SortableContext
                id={status.id}
                items={tasks.map((task) => task.id)}
                strategy={verticalListSortingStrategy}
            >
                <ul
                    ref={setNodeRef}
                    className={`flex min-h-16 flex-col gap-2 px-3 pb-3 ${isBoard ? "overflow-y-auto" : ""}`}
                >
                    {tasks.map((task) => (
                        <SortableTaskCard
                            key={task.id}
                            task={task}
                            layout={layout}
                            onAction={(action) => onTaskAction(task, action)}
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
