import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";

type ProjectHeaderProps = {
    project: Project | undefined;
    planCount: number;
    isLoading: boolean;
    isError: boolean;
};

export function ProjectHeader({
    project,
    planCount,
    isError,
    isLoading,
}: ProjectHeaderProps) {
    const t = useTranslate();

    if (isLoading)
        return <p className="text-sm text-gray-500">{t("common.loading")}</p>;
    if (isError)
        return <p className="text-sm text-red-600">{t("common.error")}</p>;

    return (
        <header className="mb-8 rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
            <h1 className="text-2xl font-bold">{project?.name}</h1>
            <p className="mt-2 text-sm text-gray-600">
                {project ? project.description : t("project.noDescription")}
            </p>
            <dl className="mt-4 flex gap-6 text-xs text-gray-500">
                <div className="flex gap-1">
                    <dt className="font-semibold text-gray-900">
                        {project?.taskCount}
                    </dt>
                    <dd>{t("workspace.projects.taskCount")}</dd>
                </div>
                <div className="flex gap-1">
                    <dd className="font-semibold text-gray-900">{planCount}</dd>
                    <dd>{t("project.plans.count")}</dd>
                </div>
            </dl>
        </header>
    );
}
