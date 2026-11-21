import type { HTMLAttributes, KeyboardEvent } from "react";
import { FiCheckCircle, FiCircle } from "react-icons/fi";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useSetTaskDone } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask, TaskPropertyKey } from "../types/task";
import { CardMenu } from "./CardMenu";
import { TaskProperties } from "./TaskProperties";

type ListRowProps = {
    task: PlanTask;
    projectId: string;
    planId: string;
    statuses: Status[];
    visible: ReadonlySet<TaskPropertyKey>;
    // The task open in the peek gets a highlight.
    isActive: boolean;
    isOverlay?: boolean;
    onAction?: (action: CardAction) => void;
    onOpen?: () => void;
};

// A row of the list layout. Sortable unless it is the drag overlay or still
// being created.
export function ListRow(props: ListRowProps) {
    const { task, isOverlay } = props;
    const isTemp = task.id.startsWith("temp-");
    const {
        attributes,
        listeners,
        setNodeRef,
        transform,
        transition,
        isDragging,
    } = useSortable({ id: task.id, disabled: isOverlay || isTemp });

    return (
        <li
            ref={isOverlay ? undefined : setNodeRef}
            style={{ transform: CSS.Translate.toString(transform), transition }}
            className={isDragging ? "opacity-40" : isTemp ? "opacity-60" : ""}
        >
            <RowContent
                {...props}
                isTemp={isTemp}
                dragProps={isOverlay || isTemp ? {} : { ...attributes, ...listeners }}
            />
        </li>
    );
}

function RowContent({
    task,
    projectId,
    planId,
    statuses,
    visible,
    isActive,
    isOverlay,
    isTemp,
    dragProps,
    onAction,
    onOpen,
}: ListRowProps & {
    isTemp: boolean;
    // dnd-kit's attributes and listeners, put on the row itself so it is the
    // keyboard drag handle (Space) while Enter opens the task.
    dragProps: HTMLAttributes<HTMLDivElement>;
}) {
    const t = useTranslate();
    const setDone = useSetTaskDone(projectId, planId);
    const isDone = task.completedAt !== null;
    const DoneIcon = isDone ? FiCheckCircle : FiCircle;
    const doneLabel = t(isDone ? "task.row.markUndone" : "task.row.markDone");

    return (
        <div
            role="button"
            tabIndex={0}
            {...dragProps}
            onClick={isTemp ? undefined : onOpen}
            onKeyDown={(event: KeyboardEvent<HTMLDivElement>) => {
                dragProps.onKeyDown?.(event);
                // Only Enter on the row itself; keys inside pills and menus
                // are theirs.
                if (event.key === "Enter" && event.target === event.currentTarget)
                    onOpen?.();
            }}
            className={`group flex min-h-11 cursor-pointer items-center gap-3 border-b border-line px-4 hover:bg-layer-1-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent ${
                isActive ? "bg-layer-1-hover ring-1 ring-accent ring-inset" : ""
            } ${isOverlay ? "bg-surface-1 shadow-overlay" : ""}`}
        >
            <button
                type="button"
                aria-pressed={isDone}
                aria-label={`${doneLabel}: ${task.title}`}
                disabled={isTemp}
                onClick={(event) => {
                    event.stopPropagation();
                    setDone.mutate({ taskId: task.id, done: !isDone });
                }}
                className={`shrink-0 cursor-pointer text-base ${
                    isDone ? "text-success" : "text-fg-placeholder hover:text-fg"
                }`}
            >
                <DoneIcon aria-hidden />
            </button>
            <span
                className={`min-w-0 flex-1 truncate text-body-xs font-medium ${
                    isDone ? "text-fg-subtle line-through" : "text-fg"
                }`}
            >
                {task.title}
            </span>
            <TaskProperties
                task={task}
                projectId={projectId}
                planId={planId}
                statuses={statuses}
                visible={visible}
            />
            {onAction && !isTemp && (
                <div className="opacity-0 focus-within:opacity-100 group-hover:opacity-100">
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
