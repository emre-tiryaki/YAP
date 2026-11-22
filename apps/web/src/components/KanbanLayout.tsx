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
import type { TaskGroup } from "../util/taskFilter";
import { KanbanColumn } from "./KanbanColumn";

type KanbanLayoutProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    groups: TaskGroup[];
    // True when tasks can only be moved between groups, not ordered.
    plain: boolean;
    // Completed tasks left out per group id.
    hiddenByGroup: Record<string, number>;
    visible: ReadonlySet<TaskPropertyKey>;
    // Only status groups can get a new status column.
    canAddStatus: boolean;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
    onCreateStatus: () => void;
    onEditStatus: (status: Status) => void;
};

export function KanbanLayout({
    projectId,
    planId,
    statuses,
    groups,
    plain,
    hiddenByGroup,
    visible,
    canAddStatus,
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
                items={groups.map((group) => group.id)}
                strategy={horizontalListSortingStrategy}
            >
                {groups.map((group) => (
                    <KanbanColumn
                        key={group.id}
                        group={group}
                        statuses={statuses}
                        projectId={projectId}
                        planId={planId}
                        plain={plain}
                        hiddenCount={hiddenByGroup[group.id] ?? 0}
                        collapsed={collapsedIds.includes(group.id)}
                        visible={visible}
                        onToggleCollapsed={() =>
                            toggleCollapsedGroup(planId, "collapsed", group.id)
                        }
                        onEdit={() => group.status && onEditStatus(group.status)}
                        onTaskAction={onTaskAction}
                        onOpenTask={onOpenTask}
                    />
                ))}
            </SortableContext>
            {canAddStatus && (
            <button
                type="button"
                onClick={onCreateStatus}
                className="flex h-12 w-72 shrink-0 cursor-pointer items-center justify-center gap-1 rounded-lg border border-dashed border-line-strong text-body-xs text-fg-subtle hover:bg-layer-1 hover:text-fg"
            >
                <FiPlus aria-hidden />
                {t("plan.status.add")}
            </button>
            )}
        </div>
    );
}
