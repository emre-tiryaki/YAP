import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";
import { ProjectCard } from "./ProjectCard";

type ProjectListProps = {
  projects: Project[];
};

export function ProjectList({ projects }: ProjectListProps) {
  const t = useTranslate();

  return (
    <section>
      <h2 className="mb-4 text-lg font-semibold">
        {t("workspace.projects.title")}
      </h2>
      {projects.length === 0 ? (
        <p className="text-sm text-gray-500">{t("workspace.projects.empty")}</p>
      ) : (
        <div className="flex flex-col gap-3">
          {projects.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}
    </section>
  );
}
