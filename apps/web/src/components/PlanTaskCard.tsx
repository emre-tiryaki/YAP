import { FiCalendar, FiCheckCircle, FiTrash2 } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import type { PlanTask } from "../types/task";
import type { BoardView } from "./ViewToggle";

type PlanTaskCardProps = {
    task: PlanTask;
    layout: BoardView;
    onDelete?: () => void;
    isOverlay?: boolean;
};

export function PlanTaskCard({
    task,
    layout,
    onDelete,
    isOverlay = false,
}: PlanTaskCardProps) {
    const t = useTranslate();
    const isDone = task.statusCategory === "done";

    return (
        <div
            className={`group flex gap-2 rounded-md border border-gray-200 bg-white p-3 ${
                layout === "board" ? "flex-col" : "items-center"
            } ${isOverlay ? "cursor-grabbing shadow-lg" : "cursor-grab shadow-sm hover:border-gray-400"}`}
        >
            <div className="flex min-w-0 flex-1 items-start gap-2">
                {isDone && (
                    <FiCheckCircle
                        aria-hidden
                        className="mt-0.5 shrink-0 text-green-600"
                    />
                )}
                <div className="min-w-0 flex-1">
                    <p
                        className={`text-sm font-medium break-words ${isDone ? "text-gray-500 line-through" : ""}`}
                    >
                        {task.title}
                    </p>
                    {layout === "board" && task.description && (
                        <p className="mt-1 line-clamp-2 text-xs text-gray-500">
                            {task.description}
                        </p>
                    )}
                </div>
                {onDelete && (
                    <button
                        type="button"
                        aria-label={t("plan.task.delete")}
                        title={t("plan.task.delete")}
                        onClick={onDelete}
                        // Keep pointer and keyboard events from starting a drag.
                        onPointerDown={(event) => event.stopPropagation()}
                        onKeyDown={(event) => event.stopPropagation()}
                        className="shrink-0 cursor-pointer rounded p-1 text-gray-400 opacity-0 group-hover:opacity-100 hover:bg-red-50 hover:text-red-600 focus:opacity-100"
                    >
                        <FiTrash2 aria-hidden />
                    </button>
                )}
            </div>
            {task.dueDate && (
                <p className="flex items-center gap-1 text-xs text-gray-500">
                    <FiCalendar aria-hidden />
                    {task.dueDate.toLocaleDateString()}
                </p>
            )}
        </div>
    );
}
