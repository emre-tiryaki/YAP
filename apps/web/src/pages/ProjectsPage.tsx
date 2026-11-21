import { useState } from "react";
import { useSearchParams } from "react-router";
import { FiGrid, FiList, FiSearch } from "react-icons/fi";
import { TopBar } from "../components/TopBar";
import { Modal } from "../components/Modal";
import { ProjectCard } from "../components/ProjectCard";
import { CreateProjectForm } from "../components/CreateProjectForm";
import { InlineError } from "../components/InlineError";
import { ViewToggle } from "../components/ViewToggle";
import { useProjects } from "../hooks/useProjects";
import { useTranslate } from "../hooks/useTranslate";
import type { BreadcrumbItem } from "../types/breadcrumb";
import type { Project } from "../types/project";
import { Button } from "../components/Button";
import { PropertySelect } from "../components/PropertySelect";

type ProjectView = "list" | "cards";
type ProjectFilter = "all" | "withTasks" | "empty";
type ProjectSort = "name" | "nameDesc" | "mostTasks" | "fewestTasks";

const filters: readonly ProjectFilter[] = ["all", "withTasks", "empty"];
const sorts: readonly ProjectSort[] = [
    "name",
    "nameDesc",
    "mostTasks",
    "fewestTasks",
];

function pick<T extends string>(value: string | null, allowed: readonly T[]): T {
    return allowed.find((item) => item === value) ?? allowed[0];
}

function matches(project: Project, query: string, filter: ProjectFilter) {
    if (filter === "withTasks" && project.taskCount === 0) return false;
    if (filter === "empty" && project.taskCount > 0) return false;
    if (!query) return true;
    return (
        project.name.toLocaleLowerCase().includes(query) ||
        project.description.toLocaleLowerCase().includes(query)
    );
}

function compare(sort: ProjectSort) {
    const byName = (a: Project, b: Project) => a.name.localeCompare(b.name);
    return (a: Project, b: Project) => {
        switch (sort) {
            case "nameDesc":
                return byName(b, a);
            case "mostTasks":
                return b.taskCount - a.taskCount || byName(a, b);
            case "fewestTasks":
                return a.taskCount - b.taskCount || byName(a, b);
            default:
                return byName(a, b);
        }
    };
}

export default function ProjectsPage() {
    const t = useTranslate();
    const projects = useProjects();
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    // Search, filter, sort and view live in the URL, so a reload or a shared
    // link keeps them. Defaults are left out to keep the URL clean.
    const [params, setParams] = useSearchParams();
    const query = params.get("q") ?? "";
    const filter = pick(params.get("filter"), filters);
    const sort = pick(params.get("sort"), sorts);
    const view = pick<ProjectView>(params.get("view"), ["list", "cards"]);

    function setParam(key: string, value: string, defaultValue: string) {
        setParams(
            (prev) => {
                const next = new URLSearchParams(prev);
                if (value === defaultValue) next.delete(key);
                else next.set(key, value);
                return next;
            },
            { replace: true },
        );
    }

    const all = projects.data ?? [];
    const needle = query.trim().toLocaleLowerCase();
    const shown = all
        .filter((project) => matches(project, needle, filter))
        .sort(compare(sort));
    const isFiltered = needle !== "" || filter !== "all";

    const path: BreadcrumbItem[] = [
        { label: t("sidebar.home"), href: "/home" },
        { label: t("workspace.projects.title"), href: "/projects" },
    ];

    return (
        <div className="flex min-w-0 flex-1 flex-col">
            <TopBar path={path} />
            <main className="flex-1 p-8">
                <div className="mb-6 flex items-center justify-between gap-4">
                    <h1 className="text-2xl font-bold">
                        {t("workspace.projects.title")}
                        {projects.data && (
                            <span className="ml-2 text-base font-normal text-gray-400 tabular-nums">
                                {all.length}
                            </span>
                        )}
                    </h1>
                    <Button variant="primary" size="lg"
                        type="button"
                        onClick={() => setIsCreateOpen(true)}
                    >
                        + {t("workspace.projects.create.title")}
                    </Button>
                </div>

                <div className="mb-6 flex flex-wrap items-center gap-3">
                    <div className="relative min-w-56 flex-1">
                        <FiSearch
                            aria-hidden
                            className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-gray-400"
                        />
                        <input
                            type="search"
                            value={query}
                            onChange={(event) =>
                                setParam("q", event.target.value, "")
                            }
                            placeholder={t("projects.search")}
                            aria-label={t("projects.search")}
                            className="w-full rounded-md border border-gray-300 bg-white py-1.5 pr-3 pl-8 text-sm"
                        />
                    </div>
                    <PropertySelect
                        variant="ghost"
                        ariaLabel={t("projects.filter.label")}
                        placeholder={t("projects.filter.label")}
                        value={filter}
                        onChange={(value) =>
                            typeof value === "string" &&
                            setParam("filter", value, "all")
                        }
                        options={filters.map((value) => ({
                            value,
                            label: t(`projects.filter.${value}`),
                        }))}
                    />
                    <PropertySelect
                        variant="ghost"
                        ariaLabel={t("projects.sort.label")}
                        placeholder={t("projects.sort.label")}
                        value={sort}
                        onChange={(value) =>
                            typeof value === "string" &&
                            setParam("sort", value, "name")
                        }
                        options={sorts.map((value) => ({
                            value,
                            label: t(`projects.sort.${value}`),
                        }))}
                    />
                    <ViewToggle<ProjectView>
                        value={view}
                        onChange={(next) => setParam("view", next, "list")}
                        options={[
                            {
                                view: "list",
                                label: t("projects.view.list"),
                                Icon: FiList,
                            },
                            {
                                view: "cards",
                                label: t("projects.view.cards"),
                                Icon: FiGrid,
                            },
                        ]}
                    />
                </div>

                {projects.isLoading && (
                    <p className="text-sm text-gray-500">
                        {t("common.loading")}
                    </p>
                )}
                {projects.isError && (
                    <InlineError
                        message={t("common.error")}
                        onRetry={() => void projects.refetch()}
                    />
                )}
                {projects.isSuccess && shown.length === 0 && (
                    <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-gray-300 p-8 text-center">
                        <p className="text-sm text-gray-600">
                            {isFiltered
                                ? t("projects.noMatch")
                                : t("workspace.projects.empty")}
                        </p>
                        {isFiltered && (
                            <button
                                type="button"
                                onClick={() =>
                                    setParams(
                                        (prev) => {
                                            const next = new URLSearchParams(prev);
                                            next.delete("q");
                                            next.delete("filter");
                                            return next;
                                        },
                                        { replace: true },
                                    )
                                }
                                className="cursor-pointer rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"
                            >
                                {t("projects.clearFilters")}
                            </button>
                        )}
                    </div>
                )}
                {shown.length > 0 &&
                    (view === "cards" ? (
                        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
                            {shown.map((project) => (
                                <ProjectCard
                                    key={project.id}
                                    project={project}
                                    layout="card"
                                />
                            ))}
                        </div>
                    ) : (
                        <div className="flex flex-col gap-3">
                            {shown.map((project) => (
                                <ProjectCard key={project.id} project={project} />
                            ))}
                        </div>
                    ))}
            </main>

            <Modal
                isOpen={isCreateOpen}
                title={t("workspace.projects.create.title")}
                onClose={() => setIsCreateOpen(false)}
            >
                <CreateProjectForm onDone={() => setIsCreateOpen(false)} />
            </Modal>
        </div>
    );
}
