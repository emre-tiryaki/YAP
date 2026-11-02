import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";

type ProjectCardProps = {
  project: Project;
};

export function ProjectCard({ project }: ProjectCardProps) {
  const t = useTranslate();

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <h3 className="font-semibold">{project.name}</h3>
      <p className="mt-1 text-sm text-gray-600">{project.description}</p>
      <p className="mt-3 text-xs text-gray-500">
        {project.taskCount} {t("workspace.projects.taskCount")}
      </p>
    </div>
  );
}
