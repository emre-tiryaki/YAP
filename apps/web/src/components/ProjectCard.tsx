import { useState } from "react";
import { Link } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Pin } from "../types/pin";
import type { Project } from "../types/project";
import { cardAccent as accent, initials } from "../util/accent";
import { setPinDragData } from "../util/pinDrag";
import { CardMenu } from "./CardMenu";
import { ProgressRing } from "./ProgressRing";
import { ProjectDialogs } from "./ProjectDialogs";

type ProjectCardProps = {
    project: Project;
    // "row" is a compact line, "card" a large tile for a grid.
    layout?: "row" | "card";
};

export function ProjectCard({ project, layout = "row" }: ProjectCardProps) {
    const t = useTranslate();
    const [action, setAction] = useState<CardAction | null>(null);
    const pin: Pin = {
        kind: "project",
        id: project.id,
        label: project.name,
        href: `/projects/${project.id}`,
    };
    const dialogs = (
        <ProjectDialogs
            project={project}
            action={action}
            onClose={() => setAction(null)}
        />
    );

    if (layout === "card")
        return (
            <div className="relative h-full">
                <Link
                    to={`/projects/${project.id}`}
                    onDragStart={(event) => setPinDragData(event, pin)}
                    className={`flex h-full flex-col rounded-xl border border-gray-200 bg-white p-5 shadow-sm transition duration-200 hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-2 focus-visible:outline-accent ${accent.hoverBorder}`}
                >
                    <span
                        aria-hidden
                        className={`grid size-14 shrink-0 place-items-center rounded-xl font-semibold ${
                            project.icon ? "text-4xl" : `text-lg ${accent.tile}`
                        }`}
                    >
                        {project.icon || initials(project.name)}
                    </span>
                    <h3
                        className="mt-4 truncate pr-6 text-lg leading-tight font-semibold text-gray-900"
                        title={project.name}
                    >
                        {project.name}
                    </h3>
                    <p
                        title={project.description || undefined}
                        className={`mt-2 line-clamp-3 min-h-15 flex-1 text-sm break-words ${
                            project.description
                                ? "text-gray-600"
                                : "text-gray-400 italic"
                        }`}
                    >
                        {project.description || t("project.noDescription")}
                    </p>
                    {project.taskCount > 0 && (
                        <div className="mt-4 flex items-center gap-3 border-t border-gray-100 pt-3 text-xs text-gray-600">
                            <ProgressRing
                                done={project.doneTaskCount}
                                total={project.taskCount}
                            />
                            <span>{t("workspace.projects.progress")}</span>
                        </div>
                    )}
                </Link>
                <div className="absolute top-3 right-3">
                    <CardMenu onSelect={setAction} pin={pin} />
                </div>
                {dialogs}
            </div>
        );

    return (
        <div className="relative">
            <Link
                to={`/projects/${project.id}`}
                onDragStart={(event) => setPinDragData(event, pin)}
                className={`flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-3 pr-12 shadow-sm transition duration-200 hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-2 focus-visible:outline-accent ${accent.hoverBorder}`}
            >
                <span
                    aria-hidden
                    className={`grid size-10 shrink-0 place-items-center rounded-lg font-semibold ${
                        project.icon ? "text-2xl" : `text-sm ${accent.tile}`
                    }`}
                >
                    {project.icon || initials(project.name)}
                </span>
                <div className="min-w-0 flex-1">
                    <h3
                        className="truncate leading-tight font-semibold text-gray-900"
                        title={project.name}
                    >
                        {project.name}
                    </h3>
                    <p
                        title={project.description || undefined}
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
                <CardMenu onSelect={setAction} pin={pin} />
            </div>
            {dialogs}
        </div>
    );
}
