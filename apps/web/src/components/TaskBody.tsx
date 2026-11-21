import { lazy, Suspense, useEffect, useState } from "react";
import { FiAlertTriangle, FiCalendar, FiCheck, FiFlag, FiTag } from "react-icons/fi";
import { Link } from "react-router";
import { ApiError } from "../api/client";
import { useChangeTaskStatus } from "../hooks/useChangeTaskStatus";
import { openPalette, setPaletteContext } from "../hooks/useCommandPalette";
import { usePlan } from "../hooks/usePlans";
import { useProject } from "../hooks/useProjects";
import { useShortcut } from "../hooks/useShortcut";
import { useSetTaskDone, useUpdateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import type { PlanTask, UpdateTaskInput } from "../types/task";
import { fromDateInputValue, toDateInputValue } from "../util/date";
import { CommentSection } from "./CommentSection";
import { DatePicker } from "./DatePicker";
import { FormError } from "./FormError";
import { Pill } from "./Pill";
import { PrioritySelect } from "./PrioritySelect";
import { PropertySelect } from "./PropertySelect";
import { Skeleton } from "./Skeleton";
import { SubtaskList } from "./SubtaskList";
import { TaskMetaRow } from "./TaskMetaRow";
import { TaskPropertyRow } from "./TaskPropertyRow";
import { TaskResourceSection } from "./TaskResourceSection";
import { TaskSection } from "./TaskSection";

// The editor pulls in TipTap, so it loads only once a task opens.
const MarkdownEditor = lazy(() =>
    import("./MarkdownEditor").then((module) => ({
        default: module.MarkdownEditor,
    })),
);

type TaskBodyProps = {
    projectId: string;
    planId: string;
    task: PlanTask;
    statuses: Status[];
    // Wide layout: the properties move to a column on the right.
    full: boolean;
};

// The content of a task, shared by the peek and the full page. Fields are
// saved as soon as they are left or changed.
export function TaskBody({ projectId, planId, task, statuses, full }: TaskBodyProps) {
    const t = useTranslate();
    const project = useProject(projectId);
    const plan = usePlan(projectId, planId);
    const update = useUpdateTask();
    const setDone = useSetTaskDone(projectId, planId);
    const changeStatus = useChangeTaskStatus(projectId, planId, statuses);
    const [title, setTitle] = useState(task.title);
    const [description, setDescription] = useState(task.description);
    // Bumped by the D shortcut to open the due date calendar.
    const [dueDateRequest, setDueDateRequest] = useState(0);
    const isDone = task.completedAt !== null;

    // The palette offers this task's actions while it is open.
    useEffect(() => {
        setPaletteContext({
            kind: "task",
            taskId: task.id,
            projectId,
            planId,
            title: task.title,
        });
        return () => setPaletteContext(null);
    }, [task.id, task.title, projectId, planId]);

    useShortcut({ key: "s", inModal: true }, () => openPalette("status"));
    useShortcut({ key: "p", inModal: true }, () => openPalette("priority"));
    useShortcut({ key: "d", inModal: true }, () => setDueDateRequest((n) => n + 1));

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

    const errorMessage =
        update.error instanceof ApiError &&
        update.error.message === "invalid_input"
            ? t("plan.task.create.invalid")
            : update.error || setDone.error || changeStatus.error
              ? t("task.detail.saveError")
              : null;

    const main = (
        <div className="flex min-w-0 flex-col gap-4">
            <p className="flex items-center gap-1 text-caption text-fg-subtle">
                <Link
                    to={`/projects/${projectId}`}
                    className="truncate hover:text-fg"
                >
                    {project.data?.name}
                </Link>
                <span aria-hidden>▸</span>
                <Link
                    to={`/projects/${projectId}/plans/${planId}`}
                    className="truncate hover:text-fg"
                >
                    {plan.data?.name}
                </Link>
            </p>

            <div className="flex items-start gap-3">
                <button
                    type="button"
                    role="checkbox"
                    aria-checked={isDone}
                    aria-label={t(isDone ? "task.detail.markOpen" : "task.detail.markDone")}
                    title={t(isDone ? "task.detail.markOpen" : "task.detail.markDone")}
                    disabled={setDone.isPending}
                    onClick={() => setDone.mutate({ taskId: task.id, done: !isDone })}
                    className={`mt-2 flex size-5 shrink-0 cursor-pointer items-center justify-center rounded-full border-2 transition-colors disabled:cursor-not-allowed ${
                        isDone
                            ? "border-success bg-success text-white"
                            : "border-line-strong text-transparent hover:border-success hover:text-success"
                    }`}
                >
                    <FiCheck aria-hidden className="size-3" />
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
                    className={`field-sizing-content w-full resize-none rounded-md border border-transparent px-2 py-1 text-xl font-semibold wrap-break-word hover:border-line focus:border-line-strong focus:outline-none ${
                        isDone ? "text-fg-subtle" : "text-fg"
                    }`}
                />
            </div>

            {!full && (
                <TaskMetaRow
                    task={task}
                    projectId={projectId}
                    planId={planId}
                    statuses={statuses}
                    dueDateRequest={dueDateRequest}
                />
            )}

            <div className="flex flex-col gap-1.5">
                <Suspense fallback={<Skeleton className="min-h-40" />}>
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

            <TaskSection title={t("task.detail.subtasks")} count={task.subtaskCount}>
                <SubtaskList taskId={task.id} />
            </TaskSection>
            <TaskSection title={t("resources.title")}>
                <TaskResourceSection taskId={task.id} />
            </TaskSection>
            <TaskSection title={t("task.detail.comments")} count={task.commentCount}>
                <CommentSection taskId={task.id} />
            </TaskSection>
        </div>
    );

    if (!full) return <div className="px-8 py-5">{main}</div>;

    return (
        <div className="grid grid-cols-1 gap-8 px-8 py-5 lg:grid-cols-[1fr_22rem]">
            {main}
            <aside
                aria-label={t("task.detail.properties")}
                className="flex flex-col gap-1 self-start lg:sticky lg:top-0"
            >
                <TaskPropertyRow
                    icon={<FiFlag />}
                    label={t("plan.task.create.status")}
                >
                    <PropertySelect
                        variant="ghost"
                        ariaLabel={t("plan.task.create.status")}
                        placeholder={t("plan.task.create.status")}
                        value={task.statusId}
                        onChange={(value) =>
                            typeof value === "string" &&
                            !changeStatus.isPending &&
                            changeStatus.change(task, value)
                        }
                        options={statuses.map((status) => ({
                            value: status.id,
                            label: status.title,
                            icon: (
                                <span
                                    aria-hidden
                                    className="size-2.5 rounded-full"
                                    style={{ backgroundColor: status.color }}
                                />
                            ),
                        }))}
                    />
                </TaskPropertyRow>
                <TaskPropertyRow
                    icon={<FiAlertTriangle />}
                    label={t("task.property.priority")}
                >
                    <PrioritySelect
                        variant="ghost"
                        value={task.priority}
                        onChange={(priority) => save({ priority })}
                    />
                </TaskPropertyRow>
                <TaskPropertyRow
                    icon={<FiTag />}
                    label={t("task.property.category")}
                >
                    <Pill>{t(`status.category.${task.statusCategory}`)}</Pill>
                </TaskPropertyRow>
                <TaskPropertyRow
                    icon={<FiCalendar />}
                    label={t("task.property.dueDate")}
                >
                    <DatePicker
                        label={t("form.dueDate")}
                        value={toDateInputValue(task.dueDate)}
                        openRequest={dueDateRequest}
                        onChange={(value) =>
                            save({ dueDate: fromDateInputValue(value) })
                        }
                    />
                </TaskPropertyRow>
            </aside>
        </div>
    );
}
