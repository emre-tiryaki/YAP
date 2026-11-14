import { useState, type DragEvent, type KeyboardEvent } from "react";
import { FiChevronRight, FiTrash2 } from "react-icons/fi";
import { LuHardDrive } from "react-icons/lu";
import { Link } from "react-router";
import { usePreferences } from "../hooks/usePreferences";
import { useTranslate } from "../hooks/useTranslate";
import type { Resource } from "../types/resource";
import { isDropAccepted, readDrop, startResourceDrag, type DroppedItems } from "../util/resourceDrag";
import { childrenOf, pathTo, type ResourceIndex } from "../util/resources";
import { ResourceIcon } from "./ResourceIcon";
import { ResourceMenu, type ResourceMenuItem } from "./ResourceMenu";

const EXPANDED_KEY = "yap.resources.expanded";

function readExpanded(): Set<string> {
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(EXPANDED_KEY) ?? "[]");
        return new Set(Array.isArray(parsed) ? parsed.filter((v) => typeof v === "string") : []);
    } catch {
        return new Set();
    }
}

type ResourceTreeProps = {
    index: ResourceIndex;
    // The open item; null for the top of the tree, undefined for the trash.
    selectedId: string | null | undefined;
    menuItems: (resource: Resource) => ResourceMenuItem[];
    onDrop: (items: NonNullable<DroppedItems>, targetId: string | null) => void;
    onRename: (resource: Resource) => void;
    onTrash: (resource: Resource) => void;
    trashCount: number;
};

// The left column: every folder and file as an expandable list.
export function ResourceTree({
    index,
    selectedId,
    menuItems,
    onDrop,
    onRename,
    onTrash,
    trashCount,
}: ResourceTreeProps) {
    const t = useTranslate();
    const { shortcutsEnabled } = usePreferences();
    const [expanded, setExpanded] = useState(readExpanded);
    const [dropTarget, setDropTarget] = useState<string | null | undefined>(undefined);
    // The folders above the open item stay open, so it is always visible.
    const forced = new Set(pathTo(index, selectedId ?? null).slice(0, -1).map((r) => r.id));

    function toggle(id: string) {
        setExpanded((prev) => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id);
            else next.add(id);
            localStorage.setItem(EXPANDED_KEY, JSON.stringify([...next]));
            return next;
        });
    }

    function dropProps(targetId: string | null) {
        return {
            onDragOver: (event: DragEvent) => {
                if (!isDropAccepted(event)) return;
                event.preventDefault();
                event.stopPropagation();
                setDropTarget(targetId);
            },
            onDragLeave: () => setDropTarget(undefined),
            onDrop: (event: DragEvent) => {
                event.preventDefault();
                event.stopPropagation();
                setDropTarget(undefined);
                const items = readDrop(event);
                if (items) onDrop(items, targetId);
            },
        };
    }

    function renderLevel(parentId: string | null, depth: number) {
        return childrenOf(index, parentId).map((r) => {
            const isOpen = expanded.has(r.id) || forced.has(r.id);
            const hasChildren = childrenOf(index, r.id).length > 0;
            return (
                <li
                    key={r.id}
                    role="treeitem"
                    aria-selected={selectedId === r.id}
                    aria-expanded={r.kind === "folder" ? isOpen : undefined}
                >
                    <div
                        draggable
                        onDragStart={(event) => startResourceDrag(event, [r.id])}
                        {...(r.kind === "folder" ? dropProps(r.id) : {})}
                        className={`group flex items-center rounded-md pr-1 text-sm ${
                            selectedId === r.id
                                ? "bg-gray-200 font-medium text-gray-900"
                                : "text-gray-700 hover:bg-gray-100"
                        } ${dropTarget === r.id ? "ring-2 ring-accent" : ""}`}
                        style={{ paddingLeft: 4 + depth * 14 }}
                    >
                        <button
                            type="button"
                            tabIndex={-1}
                            onClick={() => toggle(r.id)}
                            aria-label={t(isOpen ? "resources.tree.collapse" : "resources.tree.expand")}
                            className={`cursor-pointer rounded p-0.5 text-gray-400 hover:text-gray-900 ${
                                r.kind === "folder" && hasChildren ? "" : "invisible"
                            }`}
                        >
                            <FiChevronRight aria-hidden className={`transition-transform ${isOpen ? "rotate-90" : ""}`} />
                        </button>
                        <Link
                            to={`/resources/${r.id}`}
                            title={r.name}
                            // Opening a folder also unfolds it; the chevron folds it back.
                            onClick={() => r.kind === "folder" && !isOpen && toggle(r.id)}
                            onKeyDown={(event: KeyboardEvent) => {
                                if (!shortcutsEnabled) return;
                                if (event.key === "F2") {
                                    event.preventDefault();
                                    onRename(r);
                                } else if (event.key === "Delete") {
                                    event.preventDefault();
                                    onTrash(r);
                                }
                            }}
                            className="flex min-w-0 flex-1 items-center gap-2 py-1 pl-1 focus:outline-none focus-visible:underline"
                        >
                            <ResourceIcon resource={r} isOpen={isOpen} />
                            <span className="truncate">{r.name}</span>
                        </Link>
                        <ResourceMenu items={menuItems(r)} />
                    </div>
                    {r.kind === "folder" && isOpen && hasChildren && (
                        <ul role="group">{renderLevel(r.id, depth + 1)}</ul>
                    )}
                </li>
            );
        });
    }

    return (
        <nav aria-label={t("resources.tree.label")} className="flex min-h-0 flex-1 flex-col">
            <ul role="tree" aria-label={t("resources.tree.label")} className="flex-1 overflow-y-auto p-2">
                <li role="treeitem" aria-selected={selectedId === null} aria-expanded>
                    <Link
                        to="/resources"
                        {...dropProps(null)}
                        className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-sm font-semibold ${
                            selectedId === null ? "bg-gray-200 text-gray-900" : "text-gray-700 hover:bg-gray-100"
                        } ${dropTarget === null ? "ring-2 ring-accent" : ""}`}
                    >
                        <LuHardDrive aria-hidden className="text-gray-500" />
                        {t("resources.title")}
                    </Link>
                    <ul role="group">{renderLevel(null, 0)}</ul>
                </li>
            </ul>
            <Link
                to="/resources/trash"
                aria-current={selectedId === undefined ? "page" : undefined}
                className="m-2 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-gray-600 hover:bg-gray-100 aria-[current=page]:bg-gray-200 aria-[current=page]:font-medium"
            >
                <FiTrash2 aria-hidden />
                <span className="flex-1">{t("resources.trash")}</span>
                {trashCount > 0 && <span className="text-xs text-gray-400">{trashCount}</span>}
            </Link>
        </nav>
    );
}
