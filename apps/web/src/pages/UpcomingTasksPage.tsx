import { useState } from "react";
import { useSearchParams } from "react-router";
import { FiPlus } from "react-icons/fi";
import { HeaderAction } from "../components/HeaderAction";
import { PageHeader } from "../components/PageHeader";
import { Modal } from "../components/Modal";
import { TaskList } from "../components/TaskList";
import { CreateTaskForm } from "../components/CreateTaskForm";
import { useProjects } from "../hooks/useProjects";
import { useShortcut } from "../hooks/useShortcut";
import { useUpcomingTasks } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { BreadcrumbItem } from "../types/breadcrumb";
import { PropertySelect } from "../components/PropertySelect";
import { FilterBar } from "../components/FilterBar";
import { FilterMenu } from "../components/FilterMenu";
import { useLabels } from "../hooks/useLabels";
import { filterTasks, hasActiveFilters } from "../util/taskFilter";
import { filtersFromParams, paramsWithFilters } from "../util/taskFilterParams";

export default function UpcomingTasksPage() {
    const t = useTranslate();
    const projects = useProjects();
    const tasks = useUpcomingTasks();
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    useShortcut({ key: "n" }, () => setIsCreateOpen(true));
    // The project filter lives in the URL so a reload keeps it.
    const [params, setParams] = useSearchParams();
    const projectId = params.get("project") ?? "";

    const filters = filtersFromParams(params);
    // Labels belong to one project, so they can only be filtered on once a
    // project is chosen.
    const labels = useLabels(projectId);
    function changeFilters(next: typeof filters) {
        setParams(paramsWithFilters(params, next), { replace: true });
    }

    const projectItems = projects.data ?? [];
    const allTasks = tasks.data ?? [];
    const inProject = projectId
        ? allTasks.filter((task) => task.projectId === projectId)
        : allTasks;
    const shown = filterTasks(inProject, filters);

    const breadcrumb: BreadcrumbItem[] = [
        { label: t("sidebar.home"), to: "/home" },
        { label: t("workspace.tasks.title"), to: "/tasks" },
    ];

    return (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <PageHeader
                breadcrumb={breadcrumb}
                actions={
                    <>
                        <FilterMenu
                            filters={filters}
                            onChange={changeFilters}
                            labels={labels.data}
                        />
                        <HeaderAction
                            variant="primary"
                            icon={FiPlus}
                            label={t("workspace.tasks.create.title")}
                            onClick={() => setIsCreateOpen(true)}
                        />
                    </>
                }
            />
            <FilterBar
                filters={filters}
                onChange={changeFilters}
                labels={labels.data}
            />
            <main className="mx-auto w-full max-w-4xl flex-1 overflow-auto p-8">
                <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
                    <h1 className="text-2xl font-bold">
                        {t("workspace.tasks.title")}
                        {tasks.data && (
                            <span className="ml-2 text-base font-normal text-fg-placeholder tabular-nums">
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
                                // Labels are per project, so a new project
                                // starts without a label filter.
                                setParams(
                                    paramsWithFilters(
                                        new URLSearchParams(value ? { project: value } : {}),
                                        { ...filters, labels: [] },
                                    ),
                                    { replace: true },
                                )
                            }
                            options={[
                                { value: "", label: t("tasks.filter.allProjects") },
                                ...projectItems.map((project) => ({
                                    value: project.id,
                                    label: `${project.icon ? `${project.icon} ` : ""}${project.name}`,
                                })),
                            ]}
                        />
                    </div>
                </div>
                <TaskList
                    tasks={shown}
                    isError={tasks.isError}
                    isLoading={tasks.isLoading}
                    onCreate={() => setIsCreateOpen(true)}
                    hideHeader
                    emptyMessage={
                        projectId || hasActiveFilters(filters)
                            ? t("tasks.filter.noMatch")
                            : undefined
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
