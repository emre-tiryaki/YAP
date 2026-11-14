import { FiCheckCircle, FiCircle, FiX } from "react-icons/fi";
import { Link } from "react-router";
import { useResourceTasks, useUnlinkResource } from "../hooks/useResources";
import { useTranslate } from "../hooks/useTranslate";

// The tasks a resource is linked to, each opening its plan with the task.
export function LinkedTaskList({ resourceId, onLink }: { resourceId: string; onLink?: () => void }) {
    const t = useTranslate();
    const tasks = useResourceTasks(resourceId);
    const unlink = useUnlinkResource();

    return (
        <div className="flex flex-col gap-3">
            {tasks.isLoading && <p className="text-sm text-gray-400">{t("common.loading")}</p>}
            {tasks.isError && <p className="text-sm text-red-600">{t("common.error")}</p>}
            {tasks.data?.length === 0 && <p className="text-sm text-gray-400">{t("resources.tasks.empty")}</p>}
            {tasks.data && tasks.data.length > 0 && (
                <ul className="flex flex-col divide-y divide-gray-100 rounded-md border border-gray-200">
                    {tasks.data.map((task) => (
                        <li key={task.id} className="group flex items-center gap-3 px-3 py-2 text-sm">
                            {task.completedAt ? (
                                <FiCheckCircle aria-hidden className="shrink-0 text-green-600" />
                            ) : (
                                <FiCircle aria-hidden className="shrink-0 text-gray-400" />
                            )}
                            <Link
                                to={`/projects/${task.projectId}/plans/${task.planId}?task=${task.id}`}
                                className="min-w-0 flex-1 truncate hover:underline"
                            >
                                {task.title}
                            </Link>
                            <span className="truncate text-xs text-gray-500">{task.projectName}</span>
                            <button
                                type="button"
                                onClick={() => unlink.mutate({ taskId: task.id, resourceId })}
                                aria-label={`${t("resources.unlink")}: ${task.title}`}
                                title={t("resources.unlink")}
                                className="cursor-pointer rounded p-1 text-gray-400 opacity-0 group-hover:opacity-100 hover:bg-gray-100 hover:text-gray-900 focus:opacity-100"
                            >
                                <FiX aria-hidden />
                            </button>
                        </li>
                    ))}
                </ul>
            )}
            {onLink && (
                <button
                    type="button"
                    onClick={onLink}
                    className="cursor-pointer self-start rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-100"
                >
                    {t("resources.linkToTask")}
                </button>
            )}
        </div>
    );
}
