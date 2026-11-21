import { useState, type FormEvent } from "react";
import { FiPlus } from "react-icons/fi";
import { useCreateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";

type QuickAddTaskProps = {
    projectId: string;
    planId: string;
    status: Status;
    variant: "list" | "kanban";
    // Owned by the column or group, so its header "+" can open the field too.
    isOpen: boolean;
    onOpenChange: (isOpen: boolean) => void;
};

// An inline "+ New task" row. Enter adds the task and keeps the field open for
// the next one; Esc, or leaving the field empty, closes it.
export function QuickAddTask({
    projectId,
    planId,
    status,
    variant,
    isOpen,
    onOpenChange,
}: QuickAddTaskProps) {
    const t = useTranslate();
    const create = useCreateTask();
    const [title, setTitle] = useState("");

    function submit(event: FormEvent) {
        event.preventDefault();
        const trimmed = title.trim();
        if (!trimmed) return;
        create.mutate({
            projectId,
            statusCategory: status.category,
            input: {
                title: trimmed,
                dueDate: null,
                planId,
                statusId: status.id,
            },
        });
        setTitle("");
    }

    if (!isOpen)
        return (
            <button
                type="button"
                onClick={() => onOpenChange(true)}
                className={`flex h-9 w-full cursor-pointer items-center gap-1.5 text-body-xs ${
                    variant === "kanban"
                        ? "rounded-md px-2 text-accent hover:bg-layer-1-hover"
                        : "px-4 text-fg-subtle hover:text-fg"
                }`}
            >
                <FiPlus aria-hidden />
                {t("plan.quickAdd.add")}
            </button>
        );

    return (
        <form
            onSubmit={submit}
            className={`flex flex-col gap-1 ${
                variant === "kanban"
                    ? "rounded-lg border border-line bg-layer-2 p-3"
                    : "border-b border-line px-4 py-2"
            }`}
        >
            <input
                autoFocus
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                onKeyDown={(event) => {
                    if (event.key === "Escape") {
                        event.stopPropagation();
                        setTitle("");
                        onOpenChange(false);
                    }
                }}
                onBlur={() => {
                    if (title.trim() === "") onOpenChange(false);
                }}
                placeholder={t("plan.task.create.name")}
                aria-label={t("plan.task.create.name")}
                className="w-full bg-transparent text-body-xs font-medium text-fg outline-none placeholder:text-fg-placeholder"
            />
            <p className="text-caption text-fg-placeholder">
                {t("plan.quickAdd.hint")}
            </p>
        </form>
    );
}
