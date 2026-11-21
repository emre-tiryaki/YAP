import type { ReactNode } from "react";
import { FiCheckSquare, FiFolder, FiSquare } from "react-icons/fi";
import { TbLayoutKanban } from "react-icons/tb";
import { ResourceIcon } from "../components/ResourceIcon";
import type { TranslationKey } from "../hooks/useTranslate";
import type { SearchResults } from "../types/search";

export type SearchHit = {
    key: string;
    href: string;
    icon: ReactNode;
    label: string;
    detail: string;
    muted?: boolean;
};

export type SearchGroup = { title: string; hits: SearchHit[] };

// A project or plan icon: its emoji, or `fallback` when it has none.
function emojiOr(emoji: string, fallback: ReactNode): ReactNode {
    if (!emoji) return fallback;
    return (
        <span aria-hidden className="w-4 shrink-0 text-center leading-none">
            {emoji}
        </span>
    );
}

const iconClass = "shrink-0 text-fg-placeholder";

// Search results as groups of links, in the order projects, plans, tasks,
// resources; empty groups are left out.
export function toSearchGroups(
    results: SearchResults,
    t: (key: TranslationKey) => string,
): SearchGroup[] {
    const groups: SearchGroup[] = [
        {
            title: t("search.projects"),
            hits: results.projects.map((p) => ({
                key: `project:${p.id}`,
                href: `/projects/${p.id}`,
                icon: emojiOr(p.icon, <FiFolder aria-hidden className={iconClass} />),
                label: p.name,
                detail: "",
            })),
        },
        {
            title: t("search.plans"),
            hits: results.plans.map((p) => ({
                key: `plan:${p.id}`,
                href: `/projects/${p.projectId}/plans/${p.id}`,
                icon: emojiOr(p.icon, <TbLayoutKanban aria-hidden className={iconClass} />),
                label: p.name,
                detail: p.projectName,
            })),
        },
        {
            title: t("search.tasks"),
            hits: results.tasks.map((task) => {
                const Icon = task.done ? FiCheckSquare : FiSquare;
                // A subtask opens through its parent task.
                const open = task.parentId ?? task.id;
                return {
                    key: `task:${task.id}`,
                    href: `/projects/${task.projectId}/plans/${task.planId}?task=${open}`,
                    icon: <Icon aria-hidden className={iconClass} />,
                    label: task.title,
                    detail:
                        `${task.projectName} › ${task.planName}` +
                        (task.parentId ? ` · ${t("search.subtask")}` : ""),
                    muted: task.done,
                };
            }),
        },
        {
            title: t("search.resources"),
            hits: results.resources.map((r) => ({
                key: `resource:${r.id}`,
                href: `/resources/${r.id}`,
                icon: <ResourceIcon resource={r} />,
                label: r.name,
                detail: r.parentName ?? t("sidebar.resources"),
            })),
        },
    ];
    return groups.filter((group) => group.hits.length > 0);
}
