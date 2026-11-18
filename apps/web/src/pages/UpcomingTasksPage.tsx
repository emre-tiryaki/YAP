import { useState } from "react";
import { useSearchParams } from "react-router";
import { TopBar } from "../components/TopBar";
import { Modal } from "../components/Modal";
import { TaskList } from "../components/TaskList";
import { CreateTaskForm } from "../components/CreateTaskForm";
import { useProjects } from "../hooks/useProjects";
import { useShortcut } from "../hooks/useShortcut";
import { useUpcomingTasks } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { BreadcrumbItem } from "../types/breadcrumb";

export default function UpcomingTasksPage() {
    const t = useTranslate();
    const projects = useProjects();
    const tasks = useUpcomingTasks();
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    useShortcut({ key: "n" }, () => setIsCreateOpen(true));
    // The project filter lives in the URL so a reload keeps it.
    const [params, setParams] = useSearchParams();
    const projectId = params.get("project") ?? "";

    const projectItems = projects.data ?? [];
    const allTasks = tasks.data ?? [];
    const shown = projectId
        ? allTasks.filter((task) => task.projectId === projectId)
        : allTasks;

    const path: BreadcrumbItem[] = [
        { label: t("sidebar.home"), href: "/home" },
        { label: t("workspace.tasks.title"), href: "/tasks" },
    ];

    return (
        <div className="flex min-w-0 flex-1 flex-col">
            <TopBar path={path} />
            <main className="mx-auto w-full max-w-4xl flex-1 p-8">
                <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
                    <h1 className="text-2xl font-bold">
                        {t("workspace.tasks.title")}
                        {tasks.data && (
                            <span className="ml-2 text-base font-normal text-gray-400 tabular-nums">
                                {shown.length}
                            </span>
                        )}
                    </h1>
                    <div className="flex items-center gap-3">
                        <select
                            aria-label={t("tasks.filter.project")}
                            value={projectId}
                            onChange={(event) =>
                                setParams(
                                    event.target.value
                                        ? { project: event.target.value }
                                        : {},
                                    { replace: true },
                                )
                            }
                            className="max-w-56 rounded-md border border-gray-300 bg-white px-2.5 py-1.5 text-sm text-gray-700"
                        >
                            <option value="">{t("tasks.filter.allProjects")}</option>
                            {projectItems.map((project) => (
                                <option key={project.id} value={project.id}>
                                    {project.icon ? `${project.icon} ` : ""}
                                    {project.name}
                                </option>
                            ))}
                        </select>
                        <button
                            type="button"
                            onClick={() => setIsCreateOpen(true)}
                            className="cursor-pointer rounded-md bg-accent px-3 py-1.5 text-sm font-medium text-accent-fg hover:bg-accent/85"
                        >
                            + {t("workspace.tasks.create.title")}
                        </button>
                    </div>
                </div>
                <TaskList
                    tasks={shown}
                    isError={tasks.isError}
                    isLoading={tasks.isLoading}
                    onCreate={() => setIsCreateOpen(true)}
                    hideHeader
                    emptyMessage={
                        projectId ? t("tasks.filter.noMatch") : undefined
                    }
                />
            </main>

            <Modal
                isOpen={isCreateOpen}
                title={t("workspace.tasks.create.title")}
                onClose={() => setIsCreateOpen(false)}
            >
                <CreateTaskForm
                    projects={projectItems}
                    onDone={() => setIsCreateOpen(false)}
                />
            </Modal>
        </div>
    );
}
