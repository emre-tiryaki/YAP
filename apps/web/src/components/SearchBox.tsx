import { useId, useRef, useState, type ReactNode } from "react";
import { useNavigate } from "react-router";
import { FiCheckSquare, FiFolder, FiSearch, FiSquare } from "react-icons/fi";
import { TbLayoutKanban } from "react-icons/tb";
import { useSearch } from "../hooks/useSearch";
import { useShortcut } from "../hooks/useShortcut";
import { useTranslate } from "../hooks/useTranslate";
import type { SearchResults } from "../types/search";
import { ResourceIcon } from "./ResourceIcon";

type Hit = {
    key: string;
    href: string;
    icon: ReactNode;
    label: string;
    detail: string;
    muted?: boolean;
};

type Group = { title: string; hits: Hit[] };

function Emoji({ emoji, fallback }: { emoji: string; fallback: ReactNode }) {
    if (!emoji) return fallback;
    return (
        <span aria-hidden className="w-4 shrink-0 text-center leading-none">
            {emoji}
        </span>
    );
}

const iconClass = "shrink-0 text-gray-400";

function toGroups(
    results: SearchResults,
    t: ReturnType<typeof useTranslate>,
): Group[] {
    const groups: Group[] = [
        {
            title: t("search.projects"),
            hits: results.projects.map((p) => ({
                key: `project:${p.id}`,
                href: `/projects/${p.id}`,
                icon: <Emoji emoji={p.icon} fallback={<FiFolder aria-hidden className={iconClass} />} />,
                label: p.name,
                detail: "",
            })),
        },
        {
            title: t("search.plans"),
            hits: results.plans.map((p) => ({
                key: `plan:${p.id}`,
                href: `/projects/${p.projectId}/plans/${p.id}`,
                icon: <Emoji emoji={p.icon} fallback={<TbLayoutKanban aria-hidden className={iconClass} />} />,
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

// The workspace-wide search in the top bar. Results show under the input;
// arrow keys move through them and Enter opens the selected one.
export function SearchBox() {
    const t = useTranslate();
    const navigate = useNavigate();
    const listId = useId();
    const inputRef = useRef<HTMLInputElement>(null);
    const [query, setQuery] = useState("");
    const [isOpen, setIsOpen] = useState(false);
    const [active, setActive] = useState(0);
    const search = useSearch(query);

    useShortcut({ key: "k", mod: true }, () => {
        inputRef.current?.focus();
        inputRef.current?.select();
    });

    const groups = search.data ? toGroups(search.data, t) : [];
    const hits = groups.flatMap((group) => group.hits);
    const hasQuery = query.trim() !== "";
    const showPanel = isOpen && hasQuery;
    const activeIndex = Math.min(active, hits.length - 1);
    const optionId = (index: number) => `${listId}-${index}`;

    function close() {
        setIsOpen(false);
        setActive(0);
    }

    function open(hit: Hit) {
        navigate(hit.href);
        setQuery("");
        close();
        inputRef.current?.blur();
    }

    function handleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
        if (event.key === "Escape") {
            event.preventDefault();
            if (isOpen && hasQuery) close();
            else inputRef.current?.blur();
            return;
        }
        if (!showPanel || hits.length === 0) return;
        if (event.key === "ArrowDown") {
            event.preventDefault();
            setActive((activeIndex + 1) % hits.length);
        } else if (event.key === "ArrowUp") {
            event.preventDefault();
            setActive((activeIndex - 1 + hits.length) % hits.length);
        } else if (event.key === "Enter") {
            event.preventDefault();
            open(hits[activeIndex]);
        }
    }

    let status: string | null = null;
    if (search.isError) status = t("search.error");
    else if (search.isPending || search.isPlaceholderData) {
        if (hits.length === 0) status = t("search.loading");
    } else if (hits.length === 0) status = t("search.empty");

    let index = 0;
    return (
        <div
            className="relative w-full"
            onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget)) close();
            }}
        >
            <FiSearch
                aria-hidden
                className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-gray-400"
            />
            <input
                ref={inputRef}
                type="search"
                role="combobox"
                value={query}
                onChange={(event) => {
                    setQuery(event.target.value);
                    setIsOpen(true);
                    setActive(0);
                }}
                onFocus={() => setIsOpen(true)}
                onKeyDown={handleKeyDown}
                placeholder={t("topbar.search")}
                aria-label={t("topbar.search")}
                aria-expanded={showPanel}
                aria-controls={listId}
                aria-autocomplete="list"
                aria-activedescendant={
                    showPanel && hits.length > 0 ? optionId(activeIndex) : undefined
                }
                autoComplete="off"
                className="w-full rounded-md border border-gray-300 bg-gray-50 py-1.5 pr-3 pl-8 text-sm"
            />
            {showPanel && (
                <div className="app-opaque absolute top-full right-0 left-0 z-40 mt-1 max-h-[70vh] overflow-y-auto rounded-md border border-gray-200 bg-white py-1 shadow-lg">
                    {status && (
                        <p role="status" className="px-3 py-2 text-sm text-gray-500">
                            {status}
                        </p>
                    )}
                    <ul id={listId} role="listbox" aria-label={t("topbar.search")}>
                        {groups.map((group) => (
                            <li key={group.title} role="presentation">
                                <p
                                    role="presentation"
                                    className="px-3 pt-2 pb-1 text-xs font-semibold tracking-wide text-gray-400 uppercase"
                                >
                                    {group.title}
                                </p>
                                <ul role="group" aria-label={group.title}>
                                    {group.hits.map((hit) => {
                                        const i = index++;
                                        const isActive = i === activeIndex;
                                        return (
                                            <li
                                                key={hit.key}
                                                id={optionId(i)}
                                                role="option"
                                                aria-selected={isActive}
                                                // Keep the focus in the input so the blur
                                                // handler does not close the list first.
                                                onMouseDown={(event) => event.preventDefault()}
                                                onMouseEnter={() => setActive(i)}
                                                onClick={() => open(hit)}
                                                className={
                                                    "flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm " +
                                                    (isActive ? "bg-gray-100" : "")
                                                }
                                            >
                                                {hit.icon}
                                                <span
                                                    className={
                                                        "min-w-0 truncate " +
                                                        (hit.muted
                                                            ? "text-gray-400 line-through"
                                                            : "text-gray-900")
                                                    }
                                                >
                                                    {hit.label}
                                                </span>
                                                {hit.detail && (
                                                    <span className="ml-auto max-w-[50%] shrink-0 truncate text-xs text-gray-400">
                                                        {hit.detail}
                                                    </span>
                                                )}
                                            </li>
                                        );
                                    })}
                                </ul>
                            </li>
                        ))}
                    </ul>
                </div>
            )}
        </div>
    );
}
