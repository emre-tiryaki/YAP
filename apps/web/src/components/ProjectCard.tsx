import { useState } from "react";
import { Link } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Project } from "../types/project";
import { CardMenu } from "./CardMenu";
import { ProjectDialogs } from "./ProjectDialogs";

type ProjectCardProps = {
    project: Project;
};

export function ProjectCard({ project }: ProjectCardProps) {
    const t = useTranslate();
    const [action, setAction] = useState<CardAction | null>(null);

    return (
        <div className="relative">
            <Link
                to={`/projects/${project.id}`}
                className="block rounded-lg border border-gray-200 bg-white p-4 pr-11 shadow-sm transition hover:border-gray-400 hover:shadow-md"
            >
                <h3 className="font-semibold">{project.name}</h3>
                <p className="mt-1 text-sm text-gray-600">
                    {project.description}
                </p>
                <p className="mt-3 text-xs text-gray-500">
                    {project.taskCount} {t("workspace.projects.taskCount")}
                </p>
            </Link>
            <div className="absolute top-3 right-3">
                <CardMenu
                    onSelect={setAction}
                    pin={{
                        kind: "project",
                        id: project.id,
                        label: project.name,
                        href: `/projects/${project.id}`,
                    }}
                />
            </div>
            <ProjectDialogs
                project={project}
                action={action}
                onClose={() => setAction(null)}
            />
        </div>
    );
}
