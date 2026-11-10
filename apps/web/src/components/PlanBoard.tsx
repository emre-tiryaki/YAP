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
    horizontalListSortingStrategy,
    SortableContext,
    sortableKeyboardCoordinates,
    verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { FiPlus } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask, StatusCategory } from "../types/task";
import { PlanTaskCard } from "./PlanTaskCard";
import { StatusColumn } from "./StatusColumn";
import type { BoardView } from "./ViewToggle";

// Task IDs per status ID, in display order.
type Columns = Record<string, string[]>;

type PlanBoardProps = {
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
    onAddTask: (statusId: string) => void;
    onTaskAction: (task: PlanTask, action: CardAction) => void;
    onOpenTask: (task: PlanTask) => void;
    onCreateStatus: () => void;
    onEditStatus: (status: Status) => void;
};

function toColumns(statuses: Status[], tasks: PlanTask[]): Columns {
    const columns: Columns = {};
    for (const status of statuses) columns[status.id] = [];
    for (const task of tasks) columns[task.statusId]?.push(task.id);
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

export function PlanBoard({
    statuses,
    tasks,
    layout,
    onMove,
    onMoveStatus,
    onAddTask,
    onTaskAction,
    onOpenTask,
    onCreateStatus,
    onEditStatus,
}: PlanBoardProps) {
    const t = useTranslate();
    const sensors = useSensors(
        // A small distance keeps clicks on buttons inside cards working.
        useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
        useSensor(KeyboardSensor, {
            coordinateGetter: sortableKeyboardCoordinates,
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

    const serverColumns = toColumns(statuses, tasks);
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
        onMove({
            taskId: task.id,
            statusId: column,
            statusCategory: status.category,
            index,
        });
    }

    function handleDragCancel() {
        setActiveId(null);
        setOverride(null);
    }

    const isBoard = layout === "board";

    return (
        <DndContext
            sensors={sensors}
            collisionDetection={collisionDetection}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragEnd={handleDragEnd}
            onDragCancel={handleDragCancel}
        >
            <div
                className={
                    isBoard
                        ? "flex items-start gap-4 overflow-x-auto pb-4"
                        : "flex flex-col gap-4"
                }
            >
                <SortableContext
                    items={orderedStatuses.map((status) => status.id)}
                    strategy={
                        isBoard
                            ? horizontalListSortingStrategy
                            : verticalListSortingStrategy
                    }
                >
                    {orderedStatuses.map((status) => (
                        <StatusColumn
                            key={status.id}
                            status={status}
                            layout={layout}
                            tasks={(columns[status.id] ?? []).flatMap((id) => {
                                const task = taskById.get(id);
                                return task ? [task] : [];
                            })}
                            onAddTask={() => onAddTask(status.id)}
                            onEdit={() => onEditStatus(status)}
                            onTaskAction={onTaskAction}
                            onOpenTask={onOpenTask}
                        />
                    ))}
                </SortableContext>
                <button
                    type="button"
                    onClick={onCreateStatus}
                    className={`flex cursor-pointer items-center justify-center gap-1 rounded-lg border border-dashed border-gray-300 text-sm text-gray-500 hover:border-gray-400 hover:bg-gray-50 hover:text-gray-900 ${
                        isBoard ? "h-12 w-72 shrink-0" : "w-full py-3"
                    }`}
                >
                    <FiPlus aria-hidden />
                    {t("plan.status.add")}
                </button>
            </div>
            <DragOverlay>
                {activeTask && (
                    <PlanTaskCard
                        task={activeTask}
                        layout={layout}
                        isOverlay
                    />
                )}
            </DragOverlay>
        </DndContext>
    );
}
