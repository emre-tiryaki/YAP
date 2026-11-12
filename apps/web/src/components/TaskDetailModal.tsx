import { lazy, Suspense, useEffect, useState, type ReactNode } from "react";
import { FiCheck, FiTrash2, FiX } from "react-icons/fi";
import { ApiError } from "../api/client";
import { useMoveTask, useUpdateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import type { PlanTask, UpdateTaskInput } from "../types/task";
import { fromDateInputValue, toDateInputValue } from "../util/date";
import { CommentSection } from "./CommentSection";
import { DatePicker } from "./DatePicker";
import { FormError } from "./FormError";
import { SubtaskList } from "./SubtaskList";

// The editor pulls in TipTap, so it loads only once the dialog opens.
const MarkdownEditor = lazy(() =>
    import("./MarkdownEditor").then((module) => ({
        default: module.MarkdownEditor,
    })),
);

type TaskDetailModalProps = {
    projectId: string;
    planId: string;
    task: PlanTask;
    statuses: Status[];
    tasks: PlanTask[];
    onDelete: () => void;
    onClose: () => void;
};

// Side panel that slides in from the right with every detail of a plan task. The wide left column holds
// the content (title, description, subtasks, comments); the right column holds
// the properties. Fields are saved as soon as they are left or changed.
export function TaskDetailModal({
    projectId,
    planId,
    task,
    statuses,
    tasks,
    onDelete,
    onClose,
}: TaskDetailModalProps) {
    const t = useTranslate();
    // Set while the closing animation runs; onClose fires when it ends.
    const [isClosing, setIsClosing] = useState(false);

    useEffect(() => {
        function handleKeyDown(event: KeyboardEvent) {
            if (event.key !== "Escape") return;
            // Blurring first saves a field that is still being edited.
            if (document.activeElement instanceof HTMLElement)
                document.activeElement.blur();
            setIsClosing(true);
        }
        document.addEventListener("keydown", handleKeyDown);
        return () => document.removeEventListener("keydown", handleKeyDown);
    }, []);

    return (
        <div className="fixed inset-0 z-50">
            <div
                aria-hidden
                onClick={() => setIsClosing(true)}
                className={`absolute inset-0 bg-black/30 ${isClosing ? "animate-fade-out" : "animate-fade-in"}`}
            />
            <aside
                role="dialog"
                aria-modal="true"
                aria-label={t("task.detail.label")}
                onAnimationEnd={(event) => {
                    if (isClosing && event.target === event.currentTarget)
                        onClose();
                }}
                className={`absolute inset-y-0 right-0 flex w-full max-w-5xl flex-col bg-white shadow-2xl sm:w-[85vw] ${
                    isClosing ? "animate-drawer-out" : "animate-drawer-in"
                }`}
            >
                {/* A tab sticking out of the drawer's left edge, vertically centred; moves with the slide animation. */}
                <button
                    type="button"
                    onClick={() => setIsClosing(true)}
                    aria-label={t("common.close")}
                    title={t("common.close")}
                    className="absolute top-3 right-3 z-10 cursor-pointer rounded-full bg-white p-2 text-gray-600 shadow-md hover:text-gray-900 sm:top-1/2 sm:right-full sm:-translate-y-1/2 sm:rounded-none sm:rounded-l-xl sm:py-4 sm:pr-1.5 sm:pl-2.5 sm:shadow-[-6px_0_12px_-4px_rgb(0_0_0/0.2)]"
                >
                    <FiX aria-hidden className="size-5" />
                </button>
                {/* Keyed so local field state resets when another task opens. */}
                <TaskDetailBody
                    key={task.id}
                    projectId={projectId}
                    planId={planId}
                    task={task}
                    statuses={statuses}
                    tasks={tasks}
                    onDelete={onDelete}
                />
            </aside>
        </div>
    );
}

type TaskDetailBodyProps = Omit<TaskDetailModalProps, "onClose">;

function TaskDetailBody({
    projectId,
    planId,
    task,
    statuses,
    tasks,
    onDelete,
}: TaskDetailBodyProps) {
    const t = useTranslate();
    const update = useUpdateTask();
    const move = useMoveTask(projectId, planId);
    const [title, setTitle] = useState(task.title);
    const [description, setDescription] = useState(task.description);

    const status = statuses.find((s) => s.id === task.statusId);
    const isDone = task.statusCategory === "done";
    // The status the round check button moves the task to.
    const toggleTarget = statuses.find((s) =>
        isDone ? s.category === "todo" : s.category === "done",
    );

    function save(input: UpdateTaskInput) {
        update.mutate({ taskId: task.id, input });
    }

    function saveTitle() {
        const next = title.trim();
        if (next === "") {
            setTitle(task.title);
            return;
        }
        if (next !== task.title) save({ title: next });
    }

    function saveDescription() {
        if (description !== task.description) save({ description });
    }

    function changeStatus(statusId: string) {
        const target = statuses.find((s) => s.id === statusId);
        if (!target || statusId === task.statusId) return;
        // Appended to the end of the target column.
        const index = tasks.filter(
            (other) => other.statusId === statusId && other.id !== task.id,
        ).length;
        move.mutate({
            taskId: task.id,
            statusId,
            statusCategory: target.category,
            index,
        });
    }

    const errorMessage =
        update.error instanceof ApiError &&
        update.error.message === "invalid_input"
            ? t("plan.task.create.invalid")
            : update.error || move.error
              ? t("task.detail.saveError")
              : null;

    return (
        <div className="flex min-h-0 flex-1 flex-col overflow-y-auto md:flex-row md:overflow-hidden">
            <section className="flex min-w-0 flex-1 flex-col gap-6 px-5 py-6 md:overflow-y-auto md:px-8">
                <div className="flex items-start gap-3">
                    <button
                        type="button"
                        role="checkbox"
                        aria-checked={isDone}
                        aria-label={
                            isDone
                                ? t("task.detail.markOpen")
                                : t("task.detail.markDone")
                        }
                        title={
                            isDone
                                ? t("task.detail.markOpen")
                                : t("task.detail.markDone")
                        }
                        disabled={!toggleTarget || move.isPending}
                        onClick={() =>
                            toggleTarget && changeStatus(toggleTarget.id)
                        }
                        className={`mt-2.5 flex size-6 shrink-0 cursor-pointer items-center justify-center rounded-full border-2 transition-colors disabled:cursor-not-allowed ${
                            isDone
                                ? "border-green-600 bg-green-600 text-white"
                                : "border-gray-300 text-transparent hover:border-green-600 hover:text-green-600"
                        }`}
                    >
                        <FiCheck aria-hidden className="size-3.5" />
                    </button>
                    <textarea
                        aria-label={t("form.title")}
                        placeholder={t("task.detail.titlePlaceholder")}
                        value={title}
                        rows={1}
                        onChange={(event) =>
                            setTitle(event.target.value.replace(/\n/g, ""))
                        }
                        onBlur={saveTitle}
                        onKeyDown={(event) => {
                            if (event.key === "Enter") {
                                event.preventDefault();
                                event.currentTarget.blur();
                            }
                        }}
                        className={`field-sizing-content w-full resize-none rounded-md border border-transparent px-2 py-1 text-2xl font-semibold wrap-break-word hover:border-gray-200 focus:border-gray-300 focus:outline-none ${
                            isDone ? "text-gray-500" : "text-gray-900"
                        }`}
                    />
                </div>

                <div className="flex flex-col gap-1.5">
                    <label
                        htmlFor="task-detail-description"
                        className="text-sm font-semibold text-gray-800"
                    >
                        {t("form.description")}
                    </label>
                    <Suspense
                        fallback={
                            <div className="min-h-40 animate-pulse rounded-md border border-gray-200 bg-gray-50" />
                        }
                    >
                        <MarkdownEditor
                            id="task-detail-description"
                            placeholder={t("task.detail.descriptionPlaceholder")}
                            value={description}
                            onChange={setDescription}
                            onBlur={saveDescription}
                        />
                    </Suspense>
                </div>
                <FormError message={errorMessage} />

                <hr className="border-gray-100" />
                <SubtaskList taskId={task.id} />

                <hr className="border-gray-100" />
                <CommentSection taskId={task.id} />
            </section>

            <aside className="flex w-full shrink-0 flex-col gap-5 border-t border-gray-200 bg-gray-50 p-5 md:w-72 md:overflow-y-auto md:border-t-0 md:border-l">
                <h3 className="text-xs font-semibold tracking-wide text-gray-500 uppercase">
                    {t("task.detail.properties")}
                </h3>

                <Property label={t("plan.task.create.status")}>
                    <div className="flex items-center gap-2">
                        <span
                            aria-hidden
                            className="size-3 shrink-0 rounded-full"
                            style={{ backgroundColor: status?.color }}
                        />
                        <select
                            aria-label={t("plan.task.create.status")}
                            value={task.statusId}
                            disabled={move.isPending}
                            onChange={(event) =>
                                changeStatus(event.target.value)
                            }
                            className="w-full cursor-pointer rounded-md border border-gray-300 bg-white px-2 py-1.5 text-sm"
                        >
                            {statuses.map((s) => (
                                <option key={s.id} value={s.id}>
                                    {s.title}
                                </option>
                            ))}
                        </select>
                    </div>
                </Property>

                <Property label={t("task.detail.category")}>
                    <p className="text-sm text-gray-800">
                        {t(`status.category.${task.statusCategory}`)}
                    </p>
                </Property>

                <Property label={t("form.dueDate")}>
                    <DatePicker
                        label={t("form.dueDate")}
                        value={toDateInputValue(task.dueDate)}
                        onChange={(value) =>
                            save({ dueDate: fromDateInputValue(value) })
                        }
                    />
                    {!task.dueDate && (
                        <p className="mt-1 text-xs text-gray-500">
                            {t("task.detail.noDueDate")}
                        </p>
                    )}
                </Property>

                <div className="mt-auto border-t border-gray-200 pt-4">
                    <button
                        type="button"
                        onClick={onDelete}
                        className="flex w-full cursor-pointer items-center justify-center gap-2 rounded-md border border-red-200 px-3 py-2 text-sm text-red-600 hover:bg-red-50"
                    >
                        <FiTrash2 aria-hidden />
                        {t("common.delete")}
                    </button>
                </div>
            </aside>
        </div>
    );
}

function Property({ label, children }: { label: string; children: ReactNode }) {
    return (
        <div className="flex flex-col gap-1.5">
            <span className="text-xs font-medium text-gray-500">{label}</span>
            {children}
        </div>
    );
}
