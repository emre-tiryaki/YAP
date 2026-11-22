import { FiX } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import type { Label } from "../types/label";
import type { Status } from "../types/status";
import type { TaskFilters } from "../types/taskView";
import { priorityInfo } from "../util/priority";
import { emptyFilters, hasActiveFilters } from "../util/taskFilter";
import { Button } from "./Button";

type FilterBarProps = {
    filters: TaskFilters;
    onChange: (filters: TaskFilters) => void;
    statuses?: Status[];
    labels?: Label[];
};

// The active filters as removable pills under the page header.
export function FilterBar({
    filters,
    onChange,
    statuses = [],
    labels = [],
}: FilterBarProps) {
    const t = useTranslate();
    if (!hasActiveFilters(filters)) return null;

    const pills: { key: string; text: string; clear: () => void }[] = [];
    if (filters.status.length > 0)
        pills.push({
            key: "status",
            text: `${t("filter.status")}: ${filters.status
                .map((id) => statuses.find((s) => s.id === id)?.title ?? id)
                .join(", ")}`,
            clear: () => onChange({ ...filters, status: [] }),
        });
    if (filters.priority.length > 0)
        pills.push({
            key: "priority",
            text: `${t("filter.priority")}: ${filters.priority
                .map((p) => t(priorityInfo(p).label))
                .join(", ")}`,
            clear: () => onChange({ ...filters, priority: [] }),
        });
    if (filters.labels.length > 0)
        pills.push({
            key: "labels",
            text: `${t("filter.labels")}: ${filters.labels
                .map((id) => labels.find((l) => l.id === id)?.name ?? id)
                .join(", ")}`,
            clear: () => onChange({ ...filters, labels: [] }),
        });
    if (filters.due !== null)
        pills.push({
            key: "due",
            text: `${t("filter.due")}: ${t(`filter.due.${filters.due}`)}`,
            clear: () => onChange({ ...filters, due: null }),
        });
    if (filters.done !== "all")
        pills.push({
            key: "done",
            text: `${t("filter.done")}: ${t(`filter.done.${filters.done}`)}`,
            clear: () => onChange({ ...filters, done: "all" }),
        });

    return (
        <div className="flex flex-wrap items-center gap-1 border-b border-line px-4 py-2">
            {pills.map((pill) => (
                <span
                    key={pill.key}
                    className="inline-flex h-6 items-center gap-1 rounded-md border border-line bg-layer-2 pr-0.5 pl-2 text-caption text-fg-muted"
                >
                    {pill.text}
                    <button
                        type="button"
                        onClick={pill.clear}
                        aria-label={`${t("filter.remove")}: ${pill.text}`}
                        className="cursor-pointer rounded p-0.5 hover:bg-layer-1-hover hover:text-fg"
                    >
                        <FiX aria-hidden />
                    </button>
                </span>
            ))}
            <Button
                variant="ghost"
                size="sm"
                onClick={() => onChange(emptyFilters)}
            >
                {t("filter.clear")}
            </Button>
        </div>
    );
}
