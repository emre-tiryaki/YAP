import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";
import { ProjectCard } from "./ProjectCard";
import { SectionHeader } from "./SectionHeader";
import { ViewAllLink } from "./ViewAllLink";
import { FiFolder } from "react-icons/fi";
import { Button } from "./Button";
import { EmptyState } from "./EmptyState";

type ProjectListProps = {
    projects: Project[];
    isLoading: boolean;
    isError: boolean;
    onCreate: () => void;
    // Shows only the first `limit` projects, with a link to the projects page.
    limit?: number;
};

export function ProjectList({
    projects,
    isError,
    isLoading,
    onCreate,
    limit,
}: ProjectListProps) {
    const t = useTranslate();
    const shown = limit === undefined ? projects : projects.slice(0, limit);

    return (
        <section>
            <SectionHeader
                title={t("workspace.projects.title")}
                href="/projects"
                onCreate={onCreate}
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
                <div className="flex flex-col gap-3">
                    {shown.map((project) => (
                        <ProjectCard key={project.id} project={project} />
                    ))}
                </div>
            )}
            {shown.length < projects.length && (
                <ViewAllLink href="/projects" total={projects.length} />
            )}
        </section>
    );
}
