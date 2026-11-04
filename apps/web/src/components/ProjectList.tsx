import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";
import { ProjectCard } from "./ProjectCard";

type ProjectListProps = {
    projects: Project[];
    isLoading: boolean;
    isError: boolean;
};

export function ProjectList({
    projects,
    isError,
    isLoading,
}: ProjectListProps) {
    const t = useTranslate();

    return (
        <section>
            <h2 className="mb-4 text-lg font-semibold">
                {t("workspace.projects.title")}
            </h2>
            {isLoading && (
                <p className="text-sm text-gray-500">{t("common.error")}</p>
            )}
            {isError && (
                <p className="text-sm text-red-600">{t("common.error")}</p>
            )}
            {!isLoading && !isError && projects.length === 0 && (
                <p className="text-sm text-gray-500">
                    {t("workspace.projects.empty")}
                </p>
            )}
            {projects.length > 0 && (
                <div className="flex flex-col gap-3">
                    {projects.map((project) => (
                        <ProjectCard key={project.id} project={project} />
                    ))}
                </div>
            )}
        </section>
    );
}
