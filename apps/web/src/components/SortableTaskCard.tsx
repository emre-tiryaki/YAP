import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { CardAction } from "../types/cardAction";
import type { PlanTask } from "../types/task";
import { PlanTaskCard } from "./PlanTaskCard";
import type { BoardView } from "./ViewToggle";

type SortableTaskCardProps = {
    task: PlanTask;
    layout: BoardView;
    onAction: (action: CardAction) => void;
};

export function SortableTaskCard({
    task,
    layout,
    onAction,
}: SortableTaskCardProps) {
    const {
        attributes,
        listeners,
        setNodeRef,
        transform,
        transition,
        isDragging,
    } = useSortable({ id: task.id });

    return (
        <li
            ref={setNodeRef}
            style={{ transform: CSS.Translate.toString(transform), transition }}
            className={`rounded-md focus:outline-2 focus:outline-gray-900 ${isDragging ? "opacity-40" : ""}`}
            {...attributes}
            {...listeners}
        >
            <PlanTaskCard task={task} layout={layout} onAction={onAction} />
        </li>
    );
}
