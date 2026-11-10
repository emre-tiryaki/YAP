import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";

type ProjectHeaderProps = {
    project: Project;
    planCount: number | undefined;
};

export function ProjectHeader({ project, planCount }: ProjectHeaderProps) {
    const t = useTranslate();

    return (
        <header className="mb-8 rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
            <h1 className="text-2xl font-bold">{project.name}</h1>
            {project.description ? (
                <p className="mt-2 text-sm text-gray-600">
                    {project.description}
                </p>
            ) : (
                <p className="mt-2 text-sm text-gray-400 italic">
                    {t("project.noDescription")}
                </p>
            )}
            <dl className="mt-4 flex gap-6 text-xs text-gray-500">
                <div className="flex gap-1">
                    <dt className="font-semibold text-gray-900">
                        {project.taskCount}
                    </dt>
                    <dd>{t("workspace.projects.taskCount")}</dd>
                </div>
                <div className="flex gap-1">
                    <dt className="font-semibold text-gray-900">
                        {planCount ?? "–"}
                    </dt>
                    <dd>{t("project.plans.count")}</dd>
                </div>
            </dl>
        </header>
    );
}

export function ProjectHeaderSkeleton() {
    return (
        <div
            aria-busy="true"
            className="mb-8 animate-pulse rounded-lg border border-gray-200 bg-white p-6 shadow-sm"
        >
            <div className="h-7 w-1/3 rounded bg-gray-200" />
            <div className="mt-3 h-4 w-2/3 rounded bg-gray-100" />
            <div className="mt-5 h-3 w-40 rounded bg-gray-100" />
        </div>
    );
}
