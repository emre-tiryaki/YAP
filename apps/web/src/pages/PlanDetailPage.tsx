import { useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import { ApiError } from "../api/client";
import { CreatePlanTaskForm } from "../components/CreatePlanTaskForm";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { CreateStatusForm } from "../components/CreateStatusForm";
import { EditStatusForm } from "../components/EditStatusForm";
import { InlineError } from "../components/InlineError";
import { Modal } from "../components/Modal";
import { TaskBoardDnd } from "../components/TaskBoardDnd";
import { CalendarLayout } from "../components/CalendarLayout";
import { SpreadsheetLayout } from "../components/SpreadsheetLayout";
import { useCurrentUserSettings } from "../hooks/useUserSettings";
import { filterTasks, orderTasks } from "../util/taskFilter";
import { PlanHeader, PlanHeaderSkeleton } from "../components/PlanHeader";
import { TaskDialogs } from "../components/TaskDialogs";
import { TaskPeek } from "../components/TaskPeek";
import { FiPlus } from "react-icons/fi";
import { HeaderAction } from "../components/HeaderAction";
import { PageHeader } from "../components/PageHeader";
import type { BreadcrumbItem } from "../types/breadcrumb";
import { ViewToggle, type BoardView } from "../components/ViewToggle";
import { DisplayMenu } from "../components/DisplayMenu";
import { FilterBar } from "../components/FilterBar";
import { FilterMenu } from "../components/FilterMenu";
import { updatePlanDisplay, usePlanDisplay } from "../hooks/usePlanDisplay";
import { useLabels } from "../hooks/useLabels";
import { filtersFromParams, paramsWithFilters } from "../util/taskFilterParams";
import {
    useDeleteStatus,
    useMoveStatus,
    usePlan,
    usePlanStatuses,
} from "../hooks/usePlans";
import { useProject } from "../hooks/useProjects";
import { useMoveTask, usePlanTasks } from "../hooks/useTasks";
import { useShortcut } from "../hooks/useShortcut";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask } from "../types/task";
import NotFoundPage from "./NotFoundPage";
import { Skeleton } from "../components/Skeleton";

export default function PlanDetailPage() {
    const t = useTranslate();
    const navigate = useNavigate();
    const { projectId = "", planId = "" } = useParams();
    const [searchParams, setSearchParams] = useSearchParams();
    const viewParam = searchParams.get("view");
    const view: BoardView =
        viewParam === "list" ||
        viewParam === "calendar" ||
        viewParam === "table"
            ? viewParam
            : "board";

    // The filters live in the URL so a reload or a shared link keeps them.
    const filters = filtersFromParams(searchParams);
    const display = usePlanDisplay(planId);
    const labels = useLabels(projectId);
    const { hideDone } = useCurrentUserSettings();

    const project = useProject(projectId);
    const plan = usePlan(projectId, planId);
    const statuses = usePlanStatuses(projectId, planId);
    const tasks = usePlanTasks(projectId, planId);
    const moveTask = useMoveTask(projectId, planId);
    const moveStatus = useMoveStatus(projectId, planId);
    const deleteStatus = useDeleteStatus(projectId, planId);

    const [newTaskStatusId, setNewTaskStatusId] = useState<string | null>(
        null,
    );
    const [isStatusModalOpen, setIsStatusModalOpen] = useState(false);
    // New tasks from the keyboard go to the first status.
    const firstStatusId = statuses.data?.[0]?.id;
    useShortcut(
        { key: "n" },
        () => {
            if (firstStatusId) setNewTaskStatusId(firstStatusId);
        },
        firstStatusId !== undefined,
    );
    const [editingStatus, setEditingStatus] = useState<Status | null>(null);
    const [deletingStatus, setDeletingStatus] = useState<Status | null>(null);
    const [taskAction, setTaskAction] = useState<{
        task: PlanTask;
        action: CardAction;
    } | null>(null);
    // Looked up in the query data so the dialog always shows fresh values
    // and closes by itself once the task is gone.
    // A link may open a task directly (?task=<id>), e.g. from a resource.
    const taskParam = searchParams.get("task");
    const [openTaskId, setOpenTaskId] = useState<string | null>(taskParam);
    // The link may also arrive while this page is already open (the search
    // in the top bar), so follow changes of the parameter too.
    const [seenTaskParam, setSeenTaskParam] = useState(taskParam);
    if (taskParam !== seenTaskParam) {
        setSeenTaskParam(taskParam);
        if (taskParam) setOpenTaskId(taskParam);
    }

    if (plan.error instanceof ApiError && plan.error.status === 404)
        return <NotFoundPage />;

    const breadcrumb: BreadcrumbItem[] = [
        { label: t("sidebar.projects"), to: "/projects" },
        { label: project.data?.name ?? "", to: `/projects/${projectId}` },
        {
            label: plan.data?.name ?? "",
            to: `/projects/${projectId}/plans/${planId}`,
        },
    ].filter((item) => item.label);

    const deletingTaskCount = deletingStatus
        ? (tasks.data?.filter((task) => task.statusId === deletingStatus.id)
              .length ?? 0)
        : 0;

    function closeDeleteStatus() {
        setDeletingStatus(null);
        deleteStatus.reset();
    }

    const shownTasks = orderTasks(
        filterTasks(tasks.data ?? [], filters).filter(
            (task) => !hideDone || task.completedAt === null,
        ),
        display.orderBy,
        display.orderDir,
    );
    const boardLoading = statuses.isLoading || tasks.isLoading;
    const boardError = statuses.isError || tasks.isError;

    return (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <PageHeader
                breadcrumb={breadcrumb}
                actions={
                    <>
                    <FilterMenu
                        filters={filters}
                        onChange={(next) =>
                            setSearchParams(paramsWithFilters(searchParams, next), {
                                replace: true,
                            })
                        }
                        statuses={statuses.data}
                        labels={labels.data}
                    />
                    <DisplayMenu
                        display={display}
                        layout={view}
                        onChange={(patch) => updatePlanDisplay(planId, patch)}
                    />
                    <ViewToggle
                        iconOnly
                        value={view}
                        onChange={(next) =>
                            setSearchParams(next === "board" ? {} : { view: next }, {
                                replace: true,
                            })
                        }
                    />
                    <HeaderAction
                        variant="primary"
                        icon={FiPlus}
                        label={t("plan.task.create.title")}
                        disabled={firstStatusId === undefined}
                        onClick={() => {
                            if (firstStatusId) setNewTaskStatusId(firstStatusId);
                        }}
                    />
                    </>
                }
            />
            <FilterBar
                filters={filters}
                onChange={(next) =>
                    setSearchParams(paramsWithFilters(searchParams, next), {
                        replace: true,
                    })
                }
                statuses={statuses.data}
                labels={labels.data}
            />
            <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-auto p-8">
                {plan.isLoading && <PlanHeaderSkeleton />}
                {plan.isError && (
                    <InlineError
                        message={t("plan.loadError")}
                        onRetry={() => void plan.refetch()}
                    />
                )}
                {plan.data && (
                    <PlanHeader
                        plan={plan.data}
                        onDeleted={() =>
                            void navigate(`/projects/${projectId}`, {
                                replace: true,
                            })
                        }
                    />
                )}

                {plan.data && (
                    <>
                        {moveTask.isError && (
                            <div className="mb-4">
                                <InlineError
                                    message={t("plan.board.moveError")}
                                />
                            </div>
                        )}
                        {moveStatus.isError && (
                            <div className="mb-4">
                                <InlineError
                                    message={t("plan.board.moveStatusError")}
                                />
                            </div>
                        )}
                        {boardLoading && (
                            <div
                                aria-busy="true"
                                className="flex gap-4"
                            >
                                {[0, 1, 2].map((i) => (
                                    <Skeleton
                                        key={i}
                                        className="h-64 w-72 rounded-lg"
                                    />
                                ))}
                            </div>
                        )}
                        {boardError && (
                            <InlineError
                                message={t("plan.board.loadError")}
                                onRetry={() => {
                                    void statuses.refetch();
                                    void tasks.refetch();
                                }}
                            />
                        )}
                        {statuses.data && tasks.data && (view === "calendar" || view === "table") && (
                            (
view === "calendar" ? (
                                <CalendarLayout
                                    projectId={projectId}
                                    planId={planId}
                                    statuses={statuses.data}
                                    tasks={shownTasks}
                                    onOpenTask={(task) => setOpenTaskId(task.id)}
                                />
                            ) : (
                                <SpreadsheetLayout
                                    projectId={projectId}
                                    planId={planId}
                                    statuses={statuses.data}
                                    tasks={shownTasks}
                                    display={display}
                                    onChangeDisplay={(patch) =>
                                        updatePlanDisplay(planId, patch)
                                    }
                                    onOpenTask={(task) => setOpenTaskId(task.id)}
                                />
                            )
                            )
                        )}
                        {statuses.data && tasks.data && (view === "board" || view === "list") && (
                            <TaskBoardDnd
                                projectId={projectId}
                                planId={planId}
                                activeTaskId={openTaskId}
                                statuses={statuses.data}
                                tasks={tasks.data}
                                filters={filters}
                                display={display}
                                layout={view}
                                onMove={(move) => moveTask.mutate(move)}
                                onMoveStatus={(move) =>
                                    moveStatus.mutate(move)
                                }
                                onTaskAction={(task, action) =>
                                    setTaskAction({ task, action })
                                }
                                onCreateStatus={() =>
                                    setIsStatusModalOpen(true)
                                }
                                onOpenTask={(task) => setOpenTaskId(task.id)}
                                onEditStatus={setEditingStatus}
                            />
                        )}
                    </>
                )}
            </main>

            <Modal
                isOpen={newTaskStatusId !== null}
                title={t("plan.task.create.title")}
                onClose={() => setNewTaskStatusId(null)}
            >
                {newTaskStatusId && statuses.data && (
                    <CreatePlanTaskForm
                        projectId={projectId}
                        planId={planId}
                        statuses={statuses.data}
                        initialStatusId={newTaskStatusId}
                        onDone={() => setNewTaskStatusId(null)}
                    />
                )}
            </Modal>

            <Modal
                isOpen={isStatusModalOpen}
                title={t("plan.status.create.title")}
                onClose={() => setIsStatusModalOpen(false)}
            >
                <CreateStatusForm
                    projectId={projectId}
                    planId={planId}
                    onDone={() => setIsStatusModalOpen(false)}
                />
            </Modal>

            <Modal
                isOpen={editingStatus !== null}
                title={t("plan.status.edit.title")}
                onClose={() => setEditingStatus(null)}
            >
                {editingStatus && (
                    <EditStatusForm
                        projectId={projectId}
                        planId={planId}
                        status={editingStatus}
                        onDone={() => setEditingStatus(null)}
                        onDelete={() => {
                            setDeletingStatus(editingStatus);
                            setEditingStatus(null);
                        }}
                    />
                )}
            </Modal>

            <ConfirmDialog
                isOpen={deletingStatus !== null}
                title={t("plan.status.delete.title")}
                message={`"${deletingStatus?.title ?? ""}" — ${
                    deletingTaskCount > 0
                        ? t("plan.status.delete.messageWithTasks").replace(
                              "{count}",
                              String(deletingTaskCount),
                          )
                        : t("plan.status.delete.message")
                }`}
                confirmLabel={t("common.delete")}
                isPending={deleteStatus.isPending}
                isError={deleteStatus.isError}
                errorMessage={
                    deleteStatus.error instanceof ApiError &&
                    deleteStatus.error.message === "conflict"
                        ? t("plan.status.delete.lastDone")
                        : undefined
                }
                onConfirm={() => {
                    if (deletingStatus)
                        deleteStatus.mutate(deletingStatus.id, {
                            onSuccess: closeDeleteStatus,
                        });
                }}
                onClose={closeDeleteStatus}
            />

            {openTaskId && (
                <TaskPeek
                    projectId={projectId}
                    planId={planId}
                    taskId={openTaskId}
                    onClose={() => {
                        setOpenTaskId(null);
                        if (searchParams.has("task"))
                            setSearchParams(
                                (params) => {
                                    params.delete("task");
                                    return params;
                                },
                                { replace: true },
                            );
                    }}
                />
            )}

            {taskAction && (
                <TaskDialogs
                    task={taskAction.task}
                    action={taskAction.action}
                    onClose={() => setTaskAction(null)}
                />
            )}
        </div>
    );
}
