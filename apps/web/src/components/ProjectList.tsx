import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";
import { ProjectCard } from "./ProjectCard";
import { SectionHeader } from "./SectionHeader";

type ProjectListProps = {
    projects: Project[];
    isLoading: boolean;
    isError: boolean;
    onCreate: () => void;
};

export function ProjectList({
    projects,
    isError,
    isLoading,
    onCreate,
}: ProjectListProps) {
    const t = useTranslate();

    return (
        <section>
            <SectionHeader
                title={t("workspace.projects.title")}
                onCreate={onCreate}
            />
            {isLoading && (
                <p className="text-sm text-gray-500">{t("common.loading")}</p>
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
