import { FiPlus } from "react-icons/fi";
import {
    toggleCollapsedGroup,
    useCollapsedGroups,
} from "../hooks/useCollapsedGroups";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask, TaskPropertyKey } from "../types/task";
import { ListGroup } from "./ListGroup";

type ListLayoutProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    tasksByStatus: Record<string, PlanTask[]>;
    hiddenByStatus: Record<string, number>;
    visible: ReadonlySet<TaskPropertyKey>;
    activeTaskId: string | null;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
    onCreateStatus: () => void;
};

export function ListLayout({
    projectId,
    planId,
    statuses,
    tasksByStatus,
    hiddenByStatus,
    visible,
    activeTaskId,
    onTaskAction,
    onOpenTask,
    onCreateStatus,
}: ListLayoutProps) {
    const t = useTranslate();
    const collapsedIds = useCollapsedGroups(planId, "listCollapsed");

    return (
        <div className="min-h-0 flex-1 overflow-auto rounded-lg border border-line">
            {statuses.map((status) => (
                <ListGroup
                    key={status.id}
                    status={status}
                    statuses={statuses}
                    projectId={projectId}
                    planId={planId}
                    tasks={tasksByStatus[status.id] ?? []}
                    hiddenCount={hiddenByStatus[status.id] ?? 0}
                    collapsed={collapsedIds.includes(status.id)}
                    visible={visible}
                    activeTaskId={activeTaskId}
                    onToggleCollapsed={() =>
                        toggleCollapsedGroup(planId, "listCollapsed", status.id)
                    }
                    onTaskAction={onTaskAction}
                    onOpenTask={onOpenTask}
                />
            ))}
            <button
                type="button"
                onClick={onCreateStatus}
                className="flex h-9 w-full cursor-pointer items-center gap-1.5 px-4 text-body-xs text-fg-subtle hover:bg-layer-1 hover:text-fg"
            >
                <FiPlus aria-hidden />
                {t("plan.status.add")}
            </button>
        </div>
    );
}
