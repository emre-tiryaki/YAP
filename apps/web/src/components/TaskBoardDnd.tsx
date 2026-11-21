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
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask, StatusCategory, TaskPropertyKey } from "../types/task";
import { groupTasks } from "../util/taskGroups";
import {
    allTaskProperties,
    storedIndex,
    type TaskViewOptions,
} from "../util/taskView";
import { KanbanCard } from "./KanbanCard";
import { KanbanLayout } from "./KanbanLayout";
import { ListLayout } from "./ListLayout";
import { ListRow } from "./ListRow";
import type { BoardView } from "./ViewToggle";

// Task IDs per status ID, in display order.
type Columns = Record<string, string[]>;

type TaskBoardDndProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    tasks: PlanTask[];
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
    // Phase 1 shows every property; the display options make it configurable.
    visible?: ReadonlySet<TaskPropertyKey>;
    onCreateStatus: () => void;
    onEditStatus: (status: Status) => void;
};

// Task ids per status in display order: the server's order with the
// completed-task settings applied.
function toColumns(
    statuses: Status[],
    tasks: PlanTask[],
    view: TaskViewOptions,
): Columns {
    const columns: Columns = {};
    for (const group of groupTasks(tasks, statuses, view))
        columns[group.id] = group.tasks.map((task) => task.id);
    return columns;
}

function isStatusDrag(data: { current?: Record<string, unknown> }) {
    return data.current?.type === "status";
}

// A dragged column only collides with other columns; a dragged task uses
// the corners of columns and cards so it can land between two cards.
const collisionDetection: CollisionDetection = (args) => {
    if (!isStatusDrag(args.active.data)) return closestCorners(args);
    return closestCenter({
        ...args,
        droppableContainers: args.droppableContainers.filter((container) =>
            isStatusDrag(container.data),
        ),
    });
};

function findColumn(columns: Columns, id: UniqueIdentifier) {
    const key = String(id);
    if (key in columns) return key;
    return Object.keys(columns).find((statusId) =>
        columns[statusId].includes(key),
    );
}

export function TaskBoardDnd({
    projectId,
    planId,
    statuses,
    tasks,
    layout,
    onMove,
    onMoveStatus,
    onTaskAction,
    onOpenTask,
    activeTaskId,
    visible = allTaskProperties,
    onCreateStatus,
    onEditStatus,
}: TaskBoardDndProps) {
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
    const serverColumns = toColumns(statuses, tasks, view);
    const columns =
        override && override.basis === tasks ? override.columns : serverColumns;
    const taskById = new Map(tasks.map((task) => [task.id, task]));
    const activeTask = activeId ? taskById.get(activeId) : undefined;

    function handleDragStart({ active }: DragStartEvent) {
        if (isStatusDrag(active.data)) return;
        setActiveId(String(active.id));
        setOverride({ basis: tasks, columns });
    }

    function handleDragOver({ active, over }: DragOverEvent) {
        if (!over || isStatusDrag(active.data)) return;
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

    function handleDragEnd(event: DragEndEvent) {
        if (isStatusDrag(event.active.data)) {
            handleStatusDragEnd(event);
            return;
        }
        const { active, over } = event;
        setActiveId(null);
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

    const tasksByStatus: Record<string, PlanTask[]> = {};
    const hiddenByStatus: Record<string, number> = {};
    for (const status of orderedStatuses) {
        tasksByStatus[status.id] = (columns[status.id] ?? []).flatMap((id) => {
            const task = taskById.get(id);
            return task ? [task] : [];
        });
        hiddenByStatus[status.id] = hideDone
            ? tasks.filter(
                  (task) =>
                      task.statusId === status.id && task.completedAt !== null,
              ).length
            : 0;
    }

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
                    tasksByStatus={tasksByStatus}
                    hiddenByStatus={hiddenByStatus}
                    visible={visible}
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
                    tasksByStatus={tasksByStatus}
                    hiddenByStatus={hiddenByStatus}
                    visible={visible}
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
