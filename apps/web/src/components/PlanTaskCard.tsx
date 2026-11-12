import { FiCalendar, FiCheckCircle } from "react-icons/fi";
import { useParams } from "react-router";
import type { CardAction } from "../types/cardAction";
import type { PlanTask } from "../types/task";
import { markdownToPlainText } from "../util/markdown";
import { CardMenu } from "./CardMenu";
import type { BoardView } from "./ViewToggle";

type PlanTaskCardProps = {
    task: PlanTask;
    layout: BoardView;
    onAction?: (action: CardAction) => void;
    isOverlay?: boolean;
    // Off when the card is shown outside a sortable board (plain click to open).
    isDraggable?: boolean;
    // Where the task lives; defaults to the plan named by the route.
    projectId?: string;
    planId?: string;
};

export function PlanTaskCard({
    task,
    layout,
    onAction,
    isOverlay = false,
    isDraggable = true,
    ...location
}: PlanTaskCardProps) {
    // On the plan page the route names the project and plan the task belongs
    // to; elsewhere the caller passes them.
    const params = useParams();
    const projectId = location.projectId ?? params.projectId ?? "";
    const planId = location.planId ?? params.planId ?? "";
    const isDone = task.statusCategory === "done";

    return (
        <div
            className={`flex gap-2 rounded-md border border-gray-200 bg-white p-3 ${
                layout === "board" ? "flex-col" : "items-center"
            } ${
                isOverlay
                    ? "cursor-grabbing shadow-lg"
                    : `${isDraggable ? "cursor-grab" : "cursor-pointer"} shadow-sm hover:border-gray-400`
            }`}
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
                            {markdownToPlainText(task.description)}
                        </p>
                    )}
                </div>
                {onAction && (
                    <div className="-mt-1 -mr-1 shrink-0">
                        <CardMenu
                            onSelect={onAction}
                            pin={{
                                kind: "task",
                                id: task.id,
                                label: task.title,
                                href: `/projects/${projectId}/plans/${planId}`,
                                projectId,
                            }}
                        />
                    </div>
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
