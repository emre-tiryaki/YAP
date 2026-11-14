import { useState } from "react";
import { FiPaperclip, FiPlus, FiX } from "react-icons/fi";
import { Link } from "react-router";
import { errorKey } from "../hooks/useResourceOperations";
import { useLinkResource, useResources, useTaskResources, useUnlinkResource } from "../hooks/useResources";
import { useTranslate } from "../hooks/useTranslate";
import { indexResources, pathTo } from "../util/resources";
import { ResourceIcon } from "./ResourceIcon";
import { ResourcePickerDialog } from "./ResourcePickerDialog";

// The "Resources" part of the task detail: linked files and folders.
export function TaskResourceSection({ taskId }: { taskId: string }) {
    const t = useTranslate();
    const linked = useTaskResources(taskId);
    const all = useResources();
    const link = useLinkResource();
    const unlink = useUnlinkResource();
    const [isPicking, setIsPicking] = useState(false);
    const index = indexResources(all.data ?? []);
    const linkedIds = new Set((linked.data ?? []).map((r) => r.id));

    return (
        <section aria-labelledby={`task-resources-${taskId}`} className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <FiPaperclip aria-hidden className="text-gray-500" />
                    <h3 id={`task-resources-${taskId}`} className="text-sm font-semibold text-gray-800">
                        {t("resources.title")}
                    </h3>
                </div>
                <button
                    type="button"
                    onClick={() => setIsPicking(true)}
                    className="flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-sm text-gray-600 hover:bg-gray-100 hover:text-gray-900"
                >
                    <FiPlus aria-hidden />
                    {t("resources.addToTask")}
                </button>
            </div>
            {linked.isError && <p className="text-sm text-red-600">{t("common.error")}</p>}
            {linked.data?.length === 0 && <p className="text-sm text-gray-400">{t("resources.taskEmpty")}</p>}
            {linked.data && linked.data.length > 0 && (
                <ul className="flex flex-col gap-1">
                    {linked.data.map((r) => {
                        const location = pathTo(index, r.parentId).map((p) => p.name).join(" / ");
                        return (
                            <li key={r.id} className="group flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-gray-50">
                                <ResourceIcon resource={r} />
                                <Link
                                    to={`/resources/${r.id}`}
                                    className={`min-w-0 truncate hover:underline ${r.trashedAt ? "text-gray-400 line-through" : "text-gray-800"}`}
                                >
                                    {r.name}
                                </Link>
                                {r.trashedAt ? (
                                    <span className="shrink-0 rounded bg-gray-100 px-1.5 text-xs text-gray-500">
                                        {t("resources.inTrash")}
                                    </span>
                                ) : (
                                    location && <span className="min-w-0 truncate text-xs text-gray-400">{location}</span>
                                )}
                                <span className="flex-1" />
                                <button
                                    type="button"
                                    onClick={() => unlink.mutate({ taskId, resourceId: r.id })}
                                    aria-label={`${t("resources.unlink")}: ${r.name}`}
                                    title={t("resources.unlink")}
                                    className="cursor-pointer rounded p-1 text-gray-400 opacity-0 group-hover:opacity-100 hover:bg-gray-200 hover:text-gray-900 focus:opacity-100"
                                >
                                    <FiX aria-hidden />
                                </button>
                            </li>
                        );
                    })}
                </ul>
            )}
            {isPicking && (
                <ResourcePickerDialog
                    title={t("resources.addToTask")}
                    confirmLabel={t("resources.link.submit")}
                    index={index}
                    mode="any"
                    excludedIds={linkedIds}
                    isPending={link.isPending}
                    errorMessage={link.error ? t(errorKey(link.error)) : null}
                    onPick={(id) => {
                        if (!id) return;
                        link.mutate(
                            { taskId, resourceId: id },
                            {
                                onSuccess: () => {
                                    setIsPicking(false);
                                    link.reset();
                                },
                            },
                        );
                    }}
                    onClose={() => {
                        setIsPicking(false);
                        link.reset();
                    }}
                />
            )}
        </section>
    );
}
