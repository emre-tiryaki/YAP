import { useState, type FormEvent } from "react";
import { FiCheck, FiCheckSquare, FiPlus, FiTrash2 } from "react-icons/fi";
import {
    useCreateSubtask,
    useDeleteTask,
    useSetSubtaskDone,
    useSubtasks,
    useUpdateTask,
} from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Subtask } from "../types/task";
import { FormError } from "./FormError";
import { InlineError } from "./InlineError";

type SubtaskListProps = {
    taskId: string;
};

// Checklist of a task's subtasks: tick them off, rename them in place, remove
// them or add new ones at the bottom.
export function SubtaskList({ taskId }: SubtaskListProps) {
    const t = useTranslate();
    const subtasks = useSubtasks(taskId);
    const create = useCreateSubtask(taskId);
    const setDone = useSetSubtaskDone(taskId);
    const update = useUpdateTask();
    const remove = useDeleteTask();
    const [isAdding, setIsAdding] = useState(false);
    const [draft, setDraft] = useState("");

    const items = subtasks.data ?? [];
    const total = items.length;
    const done = items.filter((sub) => sub.statusCategory === "done").length;
    const percent = total === 0 ? 0 : Math.round((done / total) * 100);

    function handleAdd(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        const title = draft.trim();
        if (title === "") return;
        // The field stays open so several steps can be typed in a row.
        create.mutate(title, { onSuccess: () => setDraft("") });
    }

    const hasError =
        create.isError || setDone.isError || update.isError || remove.isError;

    return (
        <section
            aria-labelledby="task-subtasks-heading"
            className="flex flex-col gap-3"
        >
            <div className="flex items-center gap-3">
                <FiCheckSquare aria-hidden className="text-gray-500" />
                <h3
                    id="task-subtasks-heading"
                    className="text-sm font-semibold text-gray-800"
                >
                    {t("task.detail.subtasks")}
                </h3>
                {total > 0 && (
                    <span className="text-xs text-gray-500">
                        {t("task.detail.subtaskProgress")
                            .replace("{done}", String(done))
                            .replace("{total}", String(total))}
                    </span>
                )}
            </div>

            {total > 0 && (
                <div
                    role="progressbar"
                    aria-label={t("task.detail.subtasks")}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-valuenow={percent}
                    className="h-1.5 overflow-hidden rounded-full bg-gray-100"
                >
                    <div
                        className={`h-full rounded-full transition-all duration-300 ${percent === 100 ? "bg-green-500" : "bg-blue-500"}`}
                        style={{ width: `${percent}%` }}
                    />
                </div>
            )}

            {subtasks.isLoading && (
                <div className="flex flex-col gap-2" aria-busy="true">
                    {[0, 1].map((i) => (
                        <div
                            key={i}
                            className="h-8 animate-pulse rounded-md bg-gray-100"
                        />
                    ))}
                </div>
            )}
            {subtasks.isError && (
                <InlineError
                    message={t("task.detail.subtasksLoadError")}
                    onRetry={() => void subtasks.refetch()}
                />
            )}

            {total > 0 && (
                <ul className="flex flex-col">
                    {items.map((sub) => (
                        <SubtaskItem
                            key={sub.id}
                            subtask={sub}
                            onToggle={() =>
                                setDone.mutate({
                                    taskId: sub.id,
                                    done: sub.statusCategory !== "done",
                                })
                            }
                            onRename={(title) =>
                                update.mutate({
                                    taskId: sub.id,
                                    input: { title },
                                })
                            }
                            onDelete={() => remove.mutate(sub.id)}
                        />
                    ))}
                </ul>
            )}

            {subtasks.isSuccess && total === 0 && !isAdding && (
                <p className="text-sm text-gray-500">
                    {t("task.detail.subtasksEmpty")}
                </p>
            )}

            {isAdding ? (
                <form onSubmit={handleAdd} className="flex items-center gap-2">
                    <span
                        aria-hidden
                        className="size-4 shrink-0 rounded border border-gray-300"
                    />
                    <input
                        aria-label={t("task.detail.subtaskTitle")}
                        placeholder={t("task.detail.subtaskPlaceholder")}
                        value={draft}
                        autoFocus
                        disabled={create.isPending}
                        onChange={(event) => setDraft(event.target.value)}
                        onKeyDown={(event) => {
                            if (event.key === "Escape") {
                                // Closes only the field, not the whole dialog.
                                event.stopPropagation();
                                setDraft("");
                                setIsAdding(false);
                            }
                        }}
                        onBlur={() => {
                            if (draft.trim() === "") setIsAdding(false);
                        }}
                        className="min-w-0 flex-1 rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:border-gray-500 focus:outline-none"
                    />
                    <button
                        type="submit"
                        disabled={create.isPending || draft.trim() === ""}
                        className="cursor-pointer rounded-md bg-accent px-3 py-1.5 text-sm text-accent-fg hover:bg-accent/85 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                        {t("common.save")}
                    </button>
                </form>
            ) : (
                <button
                    type="button"
                    onClick={() => setIsAdding(true)}
                    className="flex w-fit cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-sm text-gray-600 hover:bg-gray-100 hover:text-gray-900"
                >
                    <FiPlus aria-hidden />
                    {t("task.detail.addSubtask")}
                </button>
            )}

            <FormError
                message={hasError ? t("task.detail.subtaskError") : null}
            />
        </section>
    );
}

type SubtaskItemProps = {
    subtask: Subtask;
    onToggle: () => void;
    onRename: (title: string) => void;
    onDelete: () => void;
};

function SubtaskItem({
    subtask,
    onToggle,
    onRename,
    onDelete,
}: SubtaskItemProps) {
    const t = useTranslate();
    const [title, setTitle] = useState(subtask.title);
    const isDone = subtask.statusCategory === "done";
    const toggleLabel = isDone
        ? t("task.detail.markOpen")
        : t("task.detail.markDone");

    function saveTitle() {
        const next = title.trim();
        if (next === "") {
            setTitle(subtask.title);
            return;
        }
        if (next !== subtask.title) onRename(next);
    }

    return (
        <li className="group flex items-center gap-2 rounded-md px-1 py-0.5 hover:bg-gray-50">
            <button
                type="button"
                role="checkbox"
                aria-checked={isDone}
                aria-label={toggleLabel}
                title={toggleLabel}
                onClick={onToggle}
                className={`flex size-4 shrink-0 cursor-pointer items-center justify-center rounded border transition-colors ${
                    isDone
                        ? "border-green-600 bg-green-600 text-white"
                        : "border-gray-400 bg-white hover:border-gray-600"
                }`}
            >
                {isDone && <FiCheck aria-hidden className="size-3" />}
            </button>
            <input
                aria-label={t("task.detail.subtaskTitle")}
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                onBlur={saveTitle}
                onKeyDown={(event) => {
                    if (event.key === "Enter") event.currentTarget.blur();
                    if (event.key === "Escape") {
                        event.stopPropagation();
                        setTitle(subtask.title);
                    }
                }}
                className={`min-w-0 flex-1 rounded border border-transparent bg-transparent px-1.5 py-1 text-sm hover:border-gray-200 focus:border-gray-300 focus:bg-white focus:outline-none ${
                    isDone ? "text-gray-400 line-through" : "text-gray-800"
                }`}
            />
            <button
                type="button"
                onClick={onDelete}
                aria-label={t("task.detail.deleteSubtask")}
                title={t("task.detail.deleteSubtask")}
                className="cursor-pointer rounded p-1 text-gray-400 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100 hover:bg-red-50 hover:text-red-600 focus:opacity-100"
            >
                <FiTrash2 aria-hidden />
            </button>
        </li>
    );
}
