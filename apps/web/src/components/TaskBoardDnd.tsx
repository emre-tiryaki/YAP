import { useState } from "react";
import {
    closestCenter,
    closestCorners,
    DndContext,
    DragOverlay,
    KeyboardSensor,
    PointerSensor,
    useSensor,
    useSensors,
    type DragEndEvent,
    type DragOverEvent,
    type CollisionDetection,
    type DragStartEvent,
    type UniqueIdentifier,
} from "@dnd-kit/core";
import {
    arrayMove,
    sortableKeyboardCoordinates,
} from "@dnd-kit/sortable";
import { useCurrentUserSettings } from "../hooks/useUserSettings";
import { useLabels } from "../hooks/useLabels";
import { useTranslate } from "../hooks/useTranslate";
import { useUpdateTask } from "../hooks/useTasks";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask, StatusCategory, TaskPriority } from "../types/task";
import type { DisplayOptions, TaskFilters } from "../types/taskView";
import { filterTasks, groupTasks, orderTasks, type GroupContext, type TaskGroup } from "../util/taskFilter";
import { toDateInputValue } from "../util/date";
import { storedIndex, type TaskViewOptions, arrangeTasks } from "../util/taskView";
import { KanbanCard } from "./KanbanCard";
import { KanbanLayout } from "./KanbanLayout";
import { ListLayout } from "./ListLayout";
import { ListRow } from "./ListRow";
import type { BoardView } from "./ViewToggle";

// Task IDs per group ID, in display order.
type Columns = Record<string, string[]>;

type TaskBoardDndProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    // Every task of the plan; the filters are applied here.
    tasks: PlanTask[];
    filters: TaskFilters;
    display: DisplayOptions;
    layout: BoardView;
    onMove: (move: {
        taskId: string;
        statusId: string;
        statusCategory: StatusCategory;
        index: number;
    }) => void;
    onMoveStatus: (move: { statusId: string; index: number }) => void;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
    // The task open in the peek, highlighted in the list.
    activeTaskId: string | null;
    onCreateStatus: () => void;
    onEditStatus: (status: Status) => void;
};

function isStatusDrag(data: { current?: Record<string, unknown> }) {
    return data.current?.type === "status";
}

function isGroupContainer(data: { current?: Record<string, unknown> }) {
    return data.current?.type === "status" || data.current?.type === "group";
}

function findColumn(columns: Columns, id: UniqueIdentifier) {
    const key = String(id);
    if (key in columns) return key;
    return Object.keys(columns).find((groupId) =>
        columns[groupId].includes(key),
    );
}

function addDays(days: number): Date {
    const date = new Date();
    return new Date(date.getFullYear(), date.getMonth(), date.getDate() + days);
}

export function TaskBoardDnd({
    projectId,
    planId,
    statuses,
    tasks,
    filters,
    display,
    layout,
    onMove,
    onMoveStatus,
    onTaskAction,
    onOpenTask,
    activeTaskId,
    onCreateStatus,
    onEditStatus,
}: TaskBoardDndProps) {
    const t = useTranslate();
    const update = useUpdateTask();
    const labels = useLabels(projectId);
    const sensors = useSensors(
        // A small distance keeps clicks on buttons inside cards working.
        useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
        // Space picks a task up (Enter opens it).
        useSensor(KeyboardSensor, {
            coordinateGetter: sortableKeyboardCoordinates,
            keyboardCodes: {
                start: ["Space"],
                cancel: ["Escape"],
                end: ["Space", "Enter"],
            },
        }),
    );
    const [activeId, setActiveId] = useState<string | null>(null);
    // Column layout while dragging and right after a drop. It is tied to the
    // task list it was derived from: once the query data changes (optimistic
    // update or refetch) the override is ignored, so no effect is needed to
    // clear it.
    const [override, setOverride] = useState<{
        basis: PlanTask[];
        columns: Columns;
    } | null>(null);

    // Column order right after a status drop, tied to the status list it was
    // derived from in the same way as `override`.
    const [statusOverride, setStatusOverride] = useState<{
        basis: Status[];
        order: Status[];
    } | null>(null);
    const orderedStatuses =
        statusOverride && statusOverride.basis === statuses
            ? statusOverride.order
            : statuses;

    const { hideDone, doneToBottom } = useCurrentUserSettings();
    const view: TaskViewOptions = { hideDone, doneToBottom };

    // Manual order inside status groups is the only case where a drop also
    // chooses a position; every other view only moves a task between groups.
    const plain = !(display.groupBy === "status" && display.orderBy === "manual");

    const context: GroupContext = {
        statuses: orderedStatuses,
        labels: labels.data ?? [],
        names: {
            all: t("plan.group.all"),
            noLabel: t("label.none"),
            priority: {
                urgent: t("task.priority.urgent"),
                high: t("task.priority.high"),
                medium: t("task.priority.medium"),
                low: t("task.priority.low"),
                none: t("task.priority.none"),
            },
            due: {
                overdue: t("due.group.overdue"),
                today: t("due.group.today"),
                week: t("due.group.week"),
                later: t("due.group.later"),
                none: t("due.group.none"),
            },
        },
    };
    const visibleTasks = orderTasks(
        filterTasks(tasks, filters),
        display.orderBy,
        display.orderDir,
    );
    const rawGroups = groupTasks(visibleTasks, display, context);
    const groups: TaskGroup[] = rawGroups.map((group) => ({
        ...group,
        tasks: arrangeTasks(group.tasks, view),
    }));
    const hiddenByGroup: Record<string, number> = {};
    for (const group of rawGroups)
        hiddenByGroup[group.id] = hideDone
            ? group.tasks.filter((task) => task.completedAt !== null).length
            : 0;

    const serverColumns: Columns = {};
    for (const group of groups)
        serverColumns[group.id] = group.tasks.map((task) => task.id);
    const columns =
        !plain && override && override.basis === tasks
            ? override.columns
            : serverColumns;
    const taskById = new Map(tasks.map((task) => [task.id, task]));
    const activeTask = activeId ? taskById.get(activeId) : undefined;

    // A dragged column only collides with other columns; a dragged task uses
    // the corners of columns and cards so it can land between two cards. In
    // plain views a task can only land on a group.
    const collisionDetection: CollisionDetection = (args) => {
        if (isStatusDrag(args.active.data))
            return closestCenter({
                ...args,
                droppableContainers: args.droppableContainers.filter(
                    (container) => isStatusDrag(container.data),
                ),
            });
        if (plain)
            return closestCorners({
                ...args,
                droppableContainers: args.droppableContainers.filter(
                    (container) => isGroupContainer(container.data),
                ),
            });
        return closestCorners(args);
    };

    function taskIdOf(active: { id: UniqueIdentifier; data: { current?: Record<string, unknown> } }) {
        const id = active.data.current?.taskId;
        return typeof id === "string" ? id : String(active.id);
    }

    function handleDragStart({ active }: DragStartEvent) {
        if (isStatusDrag(active.data)) return;
        setActiveId(taskIdOf(active));
        if (!plain) setOverride({ basis: tasks, columns });
    }

    function handleDragOver({ active, over }: DragOverEvent) {
        if (plain || !over || isStatusDrag(active.data)) return;
        setOverride((prev) => {
            if (!prev) return prev;
            const cols = prev.columns;
            const from = findColumn(cols, active.id);
            const to = findColumn(cols, over.id);
            if (!from || !to || from === to) return prev;

            const id = String(active.id);
            const target = cols[to];
            const overIndex = target.indexOf(String(over.id));
            const at = overIndex >= 0 ? overIndex : target.length;
            return {
                ...prev,
                columns: {
                    ...cols,
                    [from]: cols[from].filter((taskId) => taskId !== id),
                    [to]: [...target.slice(0, at), id, ...target.slice(at)],
                },
            };
        });
    }

    function handleStatusDragEnd({ active, over }: DragEndEvent) {
        const ids = orderedStatuses.map((s) => s.id);
        const oldIndex = ids.indexOf(String(active.id));
        const newIndex = over ? ids.indexOf(String(over.id)) : -1;
        if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) return;
        setStatusOverride({
            basis: statuses,
            order: arrayMove(orderedStatuses, oldIndex, newIndex),
        });
        onMoveStatus({ statusId: String(active.id), index: newIndex });
    }

    // A drop in a view that is not sorted by hand: change the property the
    // groups are made of.
    function handlePlainDragEnd({ active, over }: DragEndEvent) {
        const task = taskById.get(taskIdOf(active));
        const fromId = active.data.current?.groupId;
        if (!over || !task || typeof fromId !== "string") return;
        const toId = String(over.id);
        if (toId === fromId) return;
        const target = groups.find((g) => g.id === toId);
        if (!target) return;

        if (display.groupBy === "status" && target.status) {
            if (task.statusId === target.status.id) return;
            onMove({
                taskId: task.id,
                statusId: target.status.id,
                statusCategory: target.status.category,
                index: tasks.filter(
                    (x) => x.statusId === toId && x.id !== task.id,
                ).length,
            });
        } else if (display.groupBy === "priority") {
            const priority = toId.slice("priority:".length) as TaskPriority;
            update.mutate({ taskId: task.id, input: { priority } });
        } else if (display.groupBy === "label") {
            const from = fromId.slice("label:".length);
            const to = toId.slice("label:".length);
            const kept = task.labelIds.filter((id) => id !== from);
            const labelIds = to === "none" ? kept : [...kept, to];
            update.mutate({ taskId: task.id, input: { labelIds } });
        } else if (display.groupBy === "due") {
            const bucket = toId.slice("due:".length);
            // Overdue is not a date anyone can drop onto.
            if (bucket === "overdue") return;
            const dueDate =
                bucket === "today"
                    ? addDays(0)
                    : bucket === "week"
                      ? addDays(6)
                      : bucket === "later"
                        ? addDays(14)
                        : null;
            if (
                dueDate &&
                task.dueDate &&
                toDateInputValue(dueDate) === toDateInputValue(task.dueDate)
            )
                return;
            update.mutate({ taskId: task.id, input: { dueDate } });
        }
    }

    function handleDragEnd(event: DragEndEvent) {
        if (isStatusDrag(event.active.data)) {
            handleStatusDragEnd(event);
            return;
        }
        setActiveId(null);
        if (plain) {
            handlePlainDragEnd(event);
            return;
        }
        const { active, over } = event;
        const task = taskById.get(String(active.id));
        const column = findColumn(columns, active.id);
        const status = statuses.find((s) => s.id === column);
        if (!over || !task || !column || !status) {
            setOverride(null);
            return;
        }

        let items = columns[column];
        const oldIndex = items.indexOf(task.id);
        const newIndex =
            String(over.id) === column
                ? items.length - 1
                : items.indexOf(String(over.id));
        if (newIndex >= 0 && oldIndex !== newIndex) {
            items = arrayMove(items, oldIndex, newIndex);
        }
        const index = items.indexOf(task.id);

        if (
            task.statusId === column &&
            serverColumns[column].indexOf(task.id) === index
        ) {
            setOverride(null);
            return;
        }
        setOverride({ basis: tasks, columns: { ...columns, [column]: items } });
        // The column may leave out or reorder tasks, so the drop position is
        // mapped onto all tasks of the status in stored order.
        const siblings = tasks.filter(
            (t) => t.statusId === column && t.id !== task.id,
        );
        onMove({
            taskId: task.id,
            statusId: column,
            statusCategory: status.category,
            index: storedIndex(items, task.id, task, siblings),
        });
    }

    function handleDragCancel() {
        setActiveId(null);
        setOverride(null);
    }

    // The groups with the tasks of the (possibly overridden) columns.
    const shownGroups: TaskGroup[] = groups.map((group) => ({
        ...group,
        tasks: (columns[group.id] ?? []).flatMap((id) => {
            const task = taskById.get(id);
            return task ? [task] : [];
        }),
    }));
    const canAddStatus = display.groupBy === "status";
    const visible = new Set(display.properties);

    return (
        <DndContext
            sensors={sensors}
            collisionDetection={collisionDetection}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragEnd={handleDragEnd}
            onDragCancel={handleDragCancel}
        >
            {layout === "board" ? (
                <KanbanLayout
                    projectId={projectId}
                    planId={planId}
                    statuses={orderedStatuses}
                    groups={shownGroups}
                    plain={plain}
                    hiddenByGroup={hiddenByGroup}
                    visible={visible}
                    canAddStatus={canAddStatus}
                    onTaskAction={onTaskAction}
                    onOpenTask={onOpenTask}
                    onCreateStatus={onCreateStatus}
                    onEditStatus={onEditStatus}
                />
            ) : (
                <ListLayout
                    projectId={projectId}
                    planId={planId}
                    statuses={orderedStatuses}
                    groups={shownGroups}
                    plain={plain}
                    hiddenByGroup={hiddenByGroup}
                    visible={visible}
                    canAddStatus={canAddStatus}
                    activeTaskId={activeTaskId}
                    onTaskAction={onTaskAction}
                    onOpenTask={onOpenTask}
                    onCreateStatus={onCreateStatus}
                />
            )}
            <DragOverlay>
                {activeTask &&
                    (layout === "board" ? (
                        <KanbanCard
                            task={activeTask}
                            statuses={statuses}
                            projectId={projectId}
                            planId={planId}
                            visible={visible}
                            isOverlay
                        />
                    ) : (
                        <ul>
                            <ListRow
                                task={activeTask}
                                projectId={projectId}
                                planId={planId}
                                statuses={statuses}
                                visible={visible}
                                isActive={false}
                                isOverlay
                            />
                        </ul>
                    ))}
            </DragOverlay>
        </DndContext>
    );
}
