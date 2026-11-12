import { useState } from "react";
import { Link } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Project } from "../types/project";
import { accentFor, initials } from "../util/accent";
import { CardMenu } from "./CardMenu";
import { ProjectDialogs } from "./ProjectDialogs";

type ProjectCardProps = {
    project: Project;
};

export function ProjectCard({ project }: ProjectCardProps) {
    const t = useTranslate();
    const [action, setAction] = useState<CardAction | null>(null);
    const accent = accentFor(project.id);

    return (
        <div className="relative">
            <Link
                to={`/projects/${project.id}`}
                className={`flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-3 pr-12 shadow-sm transition duration-200 hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-2 focus-visible:outline-gray-900 ${accent.hoverBorder}`}
            >
                <span
                    aria-hidden
                    className={`grid size-10 shrink-0 place-items-center rounded-lg text-sm font-semibold ${accent.tile}`}
                >
                    {initials(project.name)}
                </span>
                <div className="min-w-0 flex-1">
                    <h3
                        className="truncate leading-tight font-semibold text-gray-900"
                        title={project.name}
                    >
                        {project.name}
                    </h3>
                    <p
                        className={`mt-0.5 truncate text-sm ${
                            project.description
                                ? "text-gray-600"
                                : "text-gray-400 italic"
                        }`}
                    >
                        {project.description || t("project.noDescription")}
                    </p>
                </div>
                <span className="shrink-0 rounded-full bg-gray-100 px-2.5 py-0.5 text-xs text-gray-600">
                    <span className="font-semibold text-gray-900 tabular-nums">
                        {project.taskCount}
                    </span>{" "}
                    {t("workspace.projects.taskCount")}
                </span>
            </Link>
            <div className="absolute top-1/2 right-3 -translate-y-1/2">
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
