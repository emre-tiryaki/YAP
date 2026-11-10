import { useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import { ApiError } from "../api/client";
import { CreatePlanTaskForm } from "../components/CreatePlanTaskForm";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { CreateStatusForm } from "../components/CreateStatusForm";
import { EditStatusForm } from "../components/EditStatusForm";
import { InlineError } from "../components/InlineError";
import { Modal } from "../components/Modal";
import { PlanBoard } from "../components/PlanBoard";
import { PlanHeader, PlanHeaderSkeleton } from "../components/PlanHeader";
import { Sidebar } from "../components/SideBar";
import { TaskDialogs } from "../components/TaskDialogs";
import { TaskDrawer } from "../components/TaskDrawer";
import { TopBar } from "../components/TopBar";
import type { BreadcrumbItem } from "../types/breadcrumb";
import { ViewToggle, type BoardView } from "../components/ViewToggle";
import {
    useDeleteStatus,
    useMoveStatus,
    usePlan,
    usePlanStatuses,
} from "../hooks/usePlans";
import { useProject } from "../hooks/useProjects";
import { useMoveTask, usePlanTasks } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Status } from "../types/status";
import type { PlanTask } from "../types/task";
import NotFoundPage from "./NotFoundPage";

export default function PlanDetailPage() {
    const t = useTranslate();
    const navigate = useNavigate();
    const { projectId = "", planId = "" } = useParams();
    const [searchParams, setSearchParams] = useSearchParams();
    const view: BoardView =
        searchParams.get("view") === "list" ? "list" : "board";

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
    const [editingStatus, setEditingStatus] = useState<Status | null>(null);
    const [deletingStatus, setDeletingStatus] = useState<Status | null>(null);
    const [taskAction, setTaskAction] = useState<{
        task: PlanTask;
        action: CardAction;
    } | null>(null);
    // Looked up in the query data so the drawer always shows fresh values
    // and closes by itself once the task is gone.
    const [openTaskId, setOpenTaskId] = useState<string | null>(null);
    const openTask = tasks.data?.find((task) => task.id === openTaskId);

    if (plan.error instanceof ApiError && plan.error.status === 404)
        return <NotFoundPage />;

    const path: BreadcrumbItem[] = [
        { label: t("sidebar.projects"), href: "/home" },
        { label: project.data?.name ?? "", href: `/projects/${projectId}` },
        {
            label: plan.data?.name ?? "",
            href: `/projects/${projectId}/plans/${planId}`,
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

    const boardLoading = statuses.isLoading || tasks.isLoading;
    const boardError = statuses.isError || tasks.isError;

    return (
        <div className="flex min-h-screen">
            <Sidebar />
            <div className="flex min-w-0 flex-1 flex-col">
                <TopBar path={path} />
                <main className="flex min-w-0 flex-1 flex-col p-8">
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
                            <div className="mb-4 flex items-center justify-between">
                                <ViewToggle
                                    value={view}
                                    onChange={(next) =>
                                        setSearchParams(
                                            next === "board"
                                                ? {}
                                                : { view: next },
                                            { replace: true },
                                        )
                                    }
                                />
                            </div>
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
                                        <div
                                            key={i}
                                            className="h-64 w-72 animate-pulse rounded-lg border border-gray-200 bg-gray-100"
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
                            {statuses.data && tasks.data && (
                                <PlanBoard
                                    statuses={statuses.data}
                                    tasks={tasks.data}
                                    layout={view}
                                    onMove={(move) => moveTask.mutate(move)}
                                    onMoveStatus={(move) =>
                                        moveStatus.mutate(move)
                                    }
                                    onAddTask={setNewTaskStatusId}
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
            </div>

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

            {openTask && statuses.data && tasks.data && (
                <TaskDrawer
                    projectId={projectId}
                    planId={planId}
                    task={openTask}
                    statuses={statuses.data}
                    tasks={tasks.data}
                    onDelete={() => {
                        setOpenTaskId(null);
                        setTaskAction({ task: openTask, action: "delete" });
                    }}
                    onClose={() => setOpenTaskId(null)}
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
