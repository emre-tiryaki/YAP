import { useParams } from "react-router";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask, TaskPropertyKey } from "../types/task";
import { useProjectIdentifier } from "../hooks/useProjects";
import { taskKey } from "../util/taskKey";
import { allTaskProperties } from "../util/taskView";
import { CardMenu } from "./CardMenu";
import { TaskProperties } from "./TaskProperties";

type KanbanCardProps = {
    task: PlanTask;
    onAction?: (action: CardAction) => void;
    isOverlay?: boolean;
    // Off when the card is shown outside a sortable board (plain click to open).
    isDraggable?: boolean;
    statuses?: Status[];
    visible?: ReadonlySet<TaskPropertyKey>;
    // Where the task lives; defaults to the plan named by the route.
    projectId?: string;
    planId?: string;
};

export function KanbanCard({
    task,
    onAction,
    isOverlay = false,
    isDraggable = true,
    statuses,
    visible = allTaskProperties,
    ...location
}: KanbanCardProps) {
    const params = useParams();
    const projectId = location.projectId ?? params.projectId ?? "";
    const planId = location.planId ?? params.planId ?? "";
    const isDone = task.completedAt !== null;
    const key = taskKey(useProjectIdentifier(projectId), task.number);

    return (
        <div
            className={`group relative rounded-lg border border-line bg-layer-2 p-3 ${
                isOverlay
                    ? "cursor-grabbing shadow-overlay"
                    : `${isDraggable ? "cursor-grab" : "cursor-pointer"} hover:border-line-strong hover:shadow-raised`
            }`}
        >
            {key && (
                <p className="text-caption text-fg-subtle">{key}</p>
            )}
            <p
                className={`pr-6 text-body-xs font-medium break-words ${
                    isDone ? "text-fg-subtle line-through" : "text-fg"
                }`}
            >
                {task.title}
            </p>
            <div className="mt-2">
                <TaskProperties
                    task={task}
                    projectId={projectId}
                    planId={planId}
                    statuses={statuses}
                    visible={visible}
                />
            </div>
            {onAction && (
                <div className="absolute top-2 right-2 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100">
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
    );
}
