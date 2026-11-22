import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";
import { ProjectCard } from "./ProjectCard";
import { SectionHeader } from "./SectionHeader";
import { ViewAllLink } from "./ViewAllLink";
import { FiFolder, FiGrid, FiList } from "react-icons/fi";
import { ViewToggle } from "./ViewToggle";
import { Button } from "./Button";
import { EmptyState } from "./EmptyState";

type ProjectListProps = {
    projects: Project[];
    isLoading: boolean;
    isError: boolean;
    onCreate: () => void;
    // Shows only the first `limit` projects, with a link to the projects page.
    limit?: number;
    view?: ProjectView;
    // Shows a list/cards switch in the header when given.
    onViewChange?: (view: ProjectView) => void;
};

export type ProjectView = "list" | "cards";

export function ProjectList({
    projects,
    isError,
    isLoading,
    onCreate,
    limit,
    view = "list",
    onViewChange,
}: ProjectListProps) {
    const t = useTranslate();
    const shown = limit === undefined ? projects : projects.slice(0, limit);

    return (
        <section className="flex min-h-0 flex-col">
            <SectionHeader
                title={t("workspace.projects.title")}
                href="/projects"
                onCreate={onCreate}
                extra={
                    onViewChange && (
                        <ViewToggle<ProjectView>
                            iconOnly
                            value={view}
                            onChange={onViewChange}
                            label={t("projects.view.label")}
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
                    )
                }
            />
            {isLoading && (
                <p className="text-sm text-gray-500">{t("common.loading")}</p>
            )}
            {isError && (
                <p className="text-sm text-red-600">{t("common.error")}</p>
            )}
            {!isLoading && !isError && projects.length === 0 && (
                <EmptyState
                    compact={limit !== undefined}
                    icon={FiFolder}
                    title={t("workspace.projects.empty")}
                    action={
                        <Button variant="primary" size="lg" onClick={onCreate}>
                            + {t("workspace.projects.create.title")}
                        </Button>
                    }
                />
            )}
            {shown.length > 0 && (
                <div
                    className={`min-h-0 flex-1 overflow-y-auto p-1 ${
                        view === "cards"
                            ? "grid grid-cols-1 content-start gap-4 sm:grid-cols-2 2xl:grid-cols-3"
                            : "flex flex-col gap-3"
                    }`}
                >
                    {shown.map((project) => (
                        <ProjectCard
                            key={project.id}
                            project={project}
                            layout={view === "cards" ? "card" : "row"}
                        />
                    ))}
                </div>
            )}
            {shown.length < projects.length && (
                <ViewAllLink href="/projects" total={projects.length} />
            )}
        </section>
    );
}
