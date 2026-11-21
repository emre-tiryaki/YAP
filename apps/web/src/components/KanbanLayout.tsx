import {
    horizontalListSortingStrategy,
    SortableContext,
} from "@dnd-kit/sortable";
import { FiPlus } from "react-icons/fi";
import {
    toggleCollapsedGroup,
    useCollapsedGroups,
} from "../hooks/useCollapsedGroups";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask, TaskPropertyKey } from "../types/task";
import { KanbanColumn } from "./KanbanColumn";

type KanbanLayoutProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    // Tasks per status id, in display order.
    tasksByStatus: Record<string, PlanTask[]>;
    // Completed tasks left out per status id.
    hiddenByStatus: Record<string, number>;
    visible: ReadonlySet<TaskPropertyKey>;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
    onCreateStatus: () => void;
    onEditStatus: (status: Status) => void;
};

export function KanbanLayout({
    projectId,
    planId,
    statuses,
    tasksByStatus,
    hiddenByStatus,
    visible,
    onTaskAction,
    onOpenTask,
    onCreateStatus,
    onEditStatus,
}: KanbanLayoutProps) {
    const t = useTranslate();
    const collapsedIds = useCollapsedGroups(planId, "collapsed");

    return (
        <div className="flex min-h-0 flex-1 items-start gap-3 overflow-x-auto pb-2">
            <SortableContext
                items={statuses.map((status) => status.id)}
                strategy={horizontalListSortingStrategy}
            >
                {statuses.map((status) => (
                    <KanbanColumn
                        key={status.id}
                        status={status}
                        statuses={statuses}
                        projectId={projectId}
                        planId={planId}
                        tasks={tasksByStatus[status.id] ?? []}
                        hiddenCount={hiddenByStatus[status.id] ?? 0}
                        collapsed={collapsedIds.includes(status.id)}
                        visible={visible}
                        onToggleCollapsed={() =>
                            toggleCollapsedGroup(planId, "collapsed", status.id)
                        }
                        onEdit={() => onEditStatus(status)}
                        onTaskAction={onTaskAction}
                        onOpenTask={onOpenTask}
                    />
                ))}
            </SortableContext>
            <button
                type="button"
                onClick={onCreateStatus}
                className="flex h-12 w-72 shrink-0 cursor-pointer items-center justify-center gap-1 rounded-lg border border-dashed border-line-strong text-body-xs text-fg-subtle hover:bg-layer-1 hover:text-fg"
            >
                <FiPlus aria-hidden />
                {t("plan.status.add")}
            </button>
        </div>
    );
}
