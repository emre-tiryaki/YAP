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
import { ListGroup } from "./ListGroup";

type ListLayoutProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    groups: TaskGroup[];
    plain: boolean;
    hiddenByGroup: Record<string, number>;
    visible: ReadonlySet<TaskPropertyKey>;
    canAddStatus: boolean;
    activeTaskId: string | null;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
    onCreateStatus: () => void;
};

export function ListLayout({
    projectId,
    planId,
    statuses,
    groups,
    plain,
    hiddenByGroup,
    visible,
    canAddStatus,
    activeTaskId,
    onTaskAction,
    onOpenTask,
    onCreateStatus,
}: ListLayoutProps) {
    const t = useTranslate();
    const collapsedIds = useCollapsedGroups(planId, "listCollapsed");

    return (
        <div className="min-h-0 flex-1 overflow-auto rounded-lg border border-line">
            {groups.map((group) => (
                <ListGroup
                    key={group.id}
                    group={group}
                    statuses={statuses}
                    projectId={projectId}
                    planId={planId}
                    plain={plain}
                    hiddenCount={hiddenByGroup[group.id] ?? 0}
                    collapsed={collapsedIds.includes(group.id)}
                    visible={visible}
                    activeTaskId={activeTaskId}
                    onToggleCollapsed={() =>
                        toggleCollapsedGroup(planId, "listCollapsed", group.id)
                    }
                    onTaskAction={onTaskAction}
                    onOpenTask={onOpenTask}
                />
            ))}
            {canAddStatus && (
            <button
                type="button"
                onClick={onCreateStatus}
                className="flex h-9 w-full cursor-pointer items-center gap-1.5 px-4 text-body-xs text-fg-subtle hover:bg-layer-1 hover:text-fg"
            >
                <FiPlus aria-hidden />
                {t("plan.status.add")}
            </button>
            )}
        </div>
    );
}
