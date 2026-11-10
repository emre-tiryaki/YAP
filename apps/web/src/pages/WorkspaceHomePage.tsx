import { useState } from "react";
import { Sidebar } from "../components/SideBar";
import { TopBar } from "../components/TopBar";
import type { BreadcrumbItem } from "../types/breadcrumb";
import { Modal } from "../components/Modal";
import { useTranslate } from "../hooks/useTranslate";
import { useWorkspace } from "../hooks/useWorkspace";
import { useUpcomingTasks } from "../hooks/useTasks";
import { useProjects } from "../hooks/useProjects";
import { ProjectList } from "../components/ProjectList";
import { CreateProjectForm } from "../components/CreateProjectForm";
import { TaskList } from "../components/TaskList";
import { CreateTaskForm } from "../components/CreateTaskForm";

export default function WorkspaceHomePage() {
    const t = useTranslate();
    const workspace = useWorkspace();
    const projects = useProjects();
    const tasks = useUpcomingTasks();
    const [isProjectModalOpen, setIsProjectModalOpen] = useState(false);
    const [isTaskModalOpen, setIsTaskModalOpen] = useState(false);

    const projectItems = projects.data ?? [];
    const path: BreadcrumbItem[] = [
        { label: t("sidebar.home"), href: "/home" },
        { label: workspace.data?.name ?? "", href: "/home" },
    ].filter((item) => item.label);

    return (
        <div className="flex min-h-screen">
            <Sidebar />
            <div className="flex flex-1 flex-col">
                <TopBar path={path} />
                <main className="flex-1 p-8">
                    <h1 className="mb-8 text-2xl font-bold">
                        {t("workspace.welcome")}
                        {workspace.data && `, ${workspace.data.username}`}
                    </h1>
                    <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
                        <ProjectList
                            projects={projectItems}
                            isError={projects.isError}
                            isLoading={projects.isLoading}
                            onCreate={() => setIsProjectModalOpen(true)}
                        />
                        <TaskList
                            tasks={tasks.data ?? []}
                            isError={tasks.isError}
                            isLoading={tasks.isLoading}
                            onCreate={() => setIsTaskModalOpen(true)}
                        />
                    </div>
                </main>
            </div>

            <Modal
                isOpen={isProjectModalOpen}
                title={t("workspace.projects.create.title")}
                onClose={() => setIsProjectModalOpen(false)}
            >
                <CreateProjectForm
                    onDone={() => setIsProjectModalOpen(false)}
                />
            </Modal>
            <Modal
                isOpen={isTaskModalOpen}
                title={t("workspace.tasks.create.title")}
                onClose={() => setIsTaskModalOpen(false)}
            >
                <CreateTaskForm
                    projects={projectItems}
                    onDone={() => setIsTaskModalOpen(false)}
                />
            </Modal>
        </div>
    );
}
