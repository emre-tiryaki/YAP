import { useState } from "react";
import { PageHeader } from "../components/PageHeader";
import type { BreadcrumbItem } from "../types/breadcrumb";
import { Modal } from "../components/Modal";
import { useShortcut } from "../hooks/useShortcut";
import { useTranslate } from "../hooks/useTranslate";
import { useWorkspace } from "../hooks/useWorkspace";
import { useUpcomingTasks } from "../hooks/useTasks";
import { useProjects } from "../hooks/useProjects";
import { ProjectList, type ProjectView } from "../components/ProjectList";
import { CreateProjectForm } from "../components/CreateProjectForm";
import { MiniCalendar } from "../components/MiniCalendar";
import { TaskList } from "../components/TaskList";
import { CreateTaskForm } from "../components/CreateTaskForm";

// The home page shows a short preview of each list; the full lists live on
// the projects and upcoming tasks pages.
const HOME_PROJECT_LIMIT = 12;
const HOME_TASK_LIMIT = 3;
const VIEW_KEY = "yap.homeProjectView";

function loadView(): ProjectView {
    try {
        return localStorage.getItem(VIEW_KEY) === "cards" ? "cards" : "list";
    } catch {
        return "list";
    }
}

export default function WorkspaceHomePage() {
    const t = useTranslate();
    const workspace = useWorkspace();
    const projects = useProjects();
    const tasks = useUpcomingTasks();
    const [isProjectModalOpen, setIsProjectModalOpen] = useState(false);
    const [isTaskModalOpen, setIsTaskModalOpen] = useState(false);
    const [view, setView] = useState<ProjectView>(loadView);
    useShortcut({ key: "n" }, () => setIsTaskModalOpen(true));

    function changeView(next: ProjectView) {
        setView(next);
        try {
            localStorage.setItem(VIEW_KEY, next);
        } catch {
            // The view just isn't remembered.
        }
    }

    const projectItems = projects.data ?? [];
    const breadcrumb: BreadcrumbItem[] = [
        { label: t("sidebar.home"), to: "/home" },
        { label: workspace.data?.name ?? "", to: "/home" },
    ].filter((item) => item.label);

    return (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <PageHeader breadcrumb={breadcrumb} />
            <main className="flex min-h-0 flex-1 flex-col overflow-hidden p-8">
                <h1 className="mb-6 shrink-0 text-2xl font-bold">
                    {t("workspace.welcome")}
                    {workspace.data && `, ${workspace.data.username}`}
                </h1>
                <div className="grid min-h-0 flex-1 grid-cols-1 gap-8 lg:grid-cols-[minmax(0,1fr)_20rem] lg:grid-rows-[minmax(0,1fr)]">
                    <ProjectList
                        projects={projectItems}
                        isError={projects.isError}
                        isLoading={projects.isLoading}
                        onCreate={() => setIsProjectModalOpen(true)}
                        limit={HOME_PROJECT_LIMIT}
                        view={view}
                        onViewChange={changeView}
                    />
                    <div className="flex min-h-0 flex-col gap-6 overflow-hidden">
                        <TaskList
                            tasks={tasks.data ?? []}
                            isError={tasks.isError}
                            isLoading={tasks.isLoading}
                            onCreate={() => setIsTaskModalOpen(true)}
                            limit={HOME_TASK_LIMIT}
                            compact
                        />
                        <MiniCalendar tasks={tasks.data ?? []} />
                    </div>
                </div>
            </main>

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
