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
import { Button } from "../components/Button";
import { PropertySelect } from "../components/PropertySelect";

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
                        <PropertySelect
                            variant="ghost"
                            ariaLabel={t("tasks.filter.project")}
                            placeholder={t("tasks.filter.allProjects")}
                            value={projectId}
                            onChange={(value) =>
                                typeof value === "string" &&
                                setParams(value ? { project: value } : {}, {
                                    replace: true,
                                })
                            }
                            options={[
                                { value: "", label: t("tasks.filter.allProjects") },
                                ...projectItems.map((project) => ({
                                    value: project.id,
                                    label: `${project.icon ? `${project.icon} ` : ""}${project.name}`,
                                })),
                            ]}
                        />
                        <Button variant="primary" size="lg"
                            type="button"
                            onClick={() => setIsCreateOpen(true)}
                        >
                            + {t("workspace.tasks.create.title")}
                        </Button>
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
