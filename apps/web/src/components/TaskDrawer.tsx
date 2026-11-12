import { lazy, Suspense, useEffect, useState, type ReactNode } from "react";
import { FiCalendar, FiCheckCircle, FiTrash2, FiX } from "react-icons/fi";
import { ApiError } from "../api/client";
import { useMoveTask, useUpdateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import type { PlanTask, UpdateTaskInput } from "../types/task";
import { fromDateInputValue, toDateInputValue } from "../util/date";
import { FormError } from "./FormError";

// The editor pulls in TipTap, so it loads only once a drawer opens.
const MarkdownEditor = lazy(() =>
    import("./MarkdownEditor").then((module) => ({
        default: module.MarkdownEditor,
    })),
);

type TaskDrawerProps = {
    projectId: string;
    planId: string;
    task: PlanTask;
    statuses: Status[];
    tasks: PlanTask[];
    onDelete: () => void;
    onClose: () => void;
};

// Side panel that slides in from the right with every detail of a plan task.
// The wide left side edits the content; the right side holds the properties.
// Changes are saved as soon as a field is left or changed.
export function TaskDrawer({
    projectId,
    planId,
    task,
    statuses,
    tasks,
    onDelete,
    onClose,
}: TaskDrawerProps) {
    const t = useTranslate();
    // Set while the slide-out animation runs; onClose fires when it ends.
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
                className={`absolute inset-y-0 right-0 flex w-full max-w-4xl flex-col bg-white shadow-2xl sm:w-[85vw] ${
                    isClosing ? "animate-drawer-out" : "animate-drawer-in"
                }`}
            >
                <header className="flex items-center justify-between border-b border-gray-200 px-6 py-3">
                    <span className="text-sm text-gray-500">
                        {t("task.detail.label")}
                    </span>
                    <button
                        type="button"
                        onClick={() => setIsClosing(true)}
                        aria-label={t("common.close")}
                        title={t("common.close")}
                        className="cursor-pointer rounded-md p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-900"
                    >
                        <FiX aria-hidden />
                    </button>
                </header>
                {/* Keyed so local field state resets when another task opens. */}
                <TaskDrawerBody
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

type TaskDrawerBodyProps = Omit<TaskDrawerProps, "onClose">;

function TaskDrawerBody({
    projectId,
    planId,
    task,
    statuses,
    tasks,
    onDelete,
}: TaskDrawerBodyProps) {
    const t = useTranslate();
    const update = useUpdateTask();
    const move = useMoveTask(projectId, planId);
    const [title, setTitle] = useState(task.title);
    const [description, setDescription] = useState(task.description);

    const status = statuses.find((s) => s.id === task.statusId);
    const isDone = task.statusCategory === "done";

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
            <section className="flex min-w-0 flex-1 flex-col gap-4 p-6 md:overflow-y-auto md:p-8">
                <div className="flex items-start gap-2">
                    {isDone && (
                        <FiCheckCircle
                            aria-hidden
                            className="mt-2 shrink-0 text-xl text-green-600"
                        />
                    )}
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
                        className="field-sizing-content w-full resize-none rounded-md border border-transparent px-2 py-1 text-2xl font-semibold wrap-break-words hover:border-gray-200 focus:border-gray-300 focus:outline-none"
                    />
                </div>
                <div className="flex flex-1 flex-col gap-1">
                    <label
                        htmlFor="task-detail-description"
                        className="px-2 text-sm font-medium text-gray-700"
                    >
                        {t("form.description")}
                    </label>
                    <Suspense
                        fallback={
                            <div className="min-h-72 flex-1 animate-pulse rounded-md border border-gray-200 bg-gray-50" />
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
            </section>

            <aside className="flex w-full shrink-0 flex-col gap-5 border-t border-gray-200 bg-gray-50 p-6 md:w-72 md:overflow-y-auto md:border-t-0 md:border-l">
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
                    <div className="flex items-center gap-2">
                        <FiCalendar aria-hidden className="text-gray-500" />
                        <input
                            type="date"
                            aria-label={t("form.dueDate")}
                            value={toDateInputValue(task.dueDate)}
                            onChange={(event) =>
                                save({
                                    dueDate: fromDateInputValue(
                                        event.target.value,
                                    ),
                                })
                            }
                            className="w-full rounded-md border border-gray-300 bg-white px-2 py-1.5 text-sm"
                        />
                    </div>
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
