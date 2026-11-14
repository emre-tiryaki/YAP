import { useState, type DragEvent, type KeyboardEvent, type MouseEvent } from "react";
import { FiArrowDown, FiArrowUp } from "react-icons/fi";
import { resourceContentUrl } from "../api/client";
import { useDateFormat } from "../hooks/useDateFormat";
import { usePreferences } from "../hooks/usePreferences";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { Resource, ResourceViewMode } from "../types/resource";
import { isDropAccepted, readDrop, startResourceDrag, type DroppedItems } from "../util/resourceDrag";
import { compareResources, fileCategory, formatSize, type ResourceSortKey } from "../util/resources";
import { ResourceIcon } from "./ResourceIcon";
import { ResourceMenu, type ResourceMenuItem } from "./ResourceMenu";

type FolderViewProps = {
    folderId: string | null;
    items: Resource[];
    mode: ResourceViewMode;
    selection: string[];
    onSelectionChange: (ids: string[]) => void;
    onOpen: (resource: Resource) => void;
    menuItems: (resource: Resource, selection: string[]) => ResourceMenuItem[];
    onDrop: (items: NonNullable<DroppedItems>, targetId: string | null) => void;
    // Keyboard commands on the selection.
    onRename: (resource: Resource) => void;
    onTrash: (ids: string[]) => void;
    onCut: (ids: string[]) => void;
    onCopy: (ids: string[]) => void;
    onPaste: () => void;
    // Ids on the clipboard for cutting, drawn faded.
    cutIds: string[];
};

const kindLabels: Record<ReturnType<typeof fileCategory>, TranslationKey> = {
    folder: "resources.kind.folder",
    image: "resources.kind.image",
    pdf: "resources.kind.pdf",
    markdown: "resources.kind.markdown",
    text: "resources.kind.text",
    audio: "resources.kind.audio",
    video: "resources.kind.video",
    other: "resources.kind.other",
};

// The contents of a folder, as a sortable list or as large icons.
export function FolderView({
    folderId,
    items,
    mode,
    selection,
    onSelectionChange,
    onOpen,
    menuItems,
    onDrop,
    onRename,
    onTrash,
    onCut,
    onCopy,
    onPaste,
    cutIds,
}: FolderViewProps) {
    const t = useTranslate();
    const format = useDateFormat();
    const { shortcutsEnabled } = usePreferences();
    const [sort, setSort] = useState<{ key: ResourceSortKey; descending: boolean }>({
        key: "name",
        descending: false,
    });
    const [anchor, setAnchor] = useState<string | null>(null);
    const [dropTarget, setDropTarget] = useState<string | null | undefined>(undefined);

    const sorted = [...items].sort((a, b) => compareResources(a, b, sort.key, sort.descending));
    const selected = new Set(selection);

    function select(event: MouseEvent, r: Resource) {
        event.stopPropagation();
        if (event.shiftKey && anchor) {
            const from = sorted.findIndex((x) => x.id === anchor);
            const to = sorted.findIndex((x) => x.id === r.id);
            const [lo, hi] = from < to ? [from, to] : [to, from];
            onSelectionChange(sorted.slice(lo, hi + 1).map((x) => x.id));
            return;
        }
        setAnchor(r.id);
        if (event.ctrlKey || event.metaKey) {
            onSelectionChange(selected.has(r.id) ? selection.filter((id) => id !== r.id) : [...selection, r.id]);
        } else {
            onSelectionChange([r.id]);
        }
    }

    function handleKeyDown(event: KeyboardEvent) {
        const mod = event.ctrlKey || event.metaKey;
        const current = sorted.filter((r) => selected.has(r.id));
        const key = event.key.toLowerCase();
        // File commands follow the shortcut setting; moving around does not.
        const commands = shortcutsEnabled;
        if (commands && event.key === "F2" && current.length === 1) onRename(current[0]);
        else if (commands && event.key === "Delete" && current.length > 0) onTrash(current.map((r) => r.id));
        else if (event.key === "Enter" && current.length === 1) onOpen(current[0]);
        else if (mod && key === "a") onSelectionChange(sorted.map((r) => r.id));
        else if (commands && mod && key === "c" && current.length > 0) onCopy(current.map((r) => r.id));
        else if (commands && mod && key === "x" && current.length > 0) onCut(current.map((r) => r.id));
        else if (commands && mod && key === "v") onPaste();
        else if (event.key === "Escape") onSelectionChange([]);
        else if (["ArrowDown", "ArrowUp", "ArrowLeft", "ArrowRight"].includes(event.key)) {
            if (sorted.length === 0) return;
            const back = event.key === "ArrowUp" || event.key === "ArrowLeft";
            const last = current.at(-1);
            const at = last ? sorted.indexOf(last) : back ? sorted.length : -1;
            const next = sorted[Math.min(sorted.length - 1, Math.max(0, at + (back ? -1 : 1)))];
            setAnchor(next.id);
            onSelectionChange([next.id]);
            document.getElementById(`resource-${next.id}`)?.scrollIntoView({ block: "nearest" });
        } else return;
        event.preventDefault();
    }

    function itemDrag(r: Resource) {
        return {
            draggable: true,
            onDragStart: (event: DragEvent) =>
                startResourceDrag(event, selected.has(r.id) ? selection : [r.id]),
            ...(r.kind === "folder"
                ? {
                      onDragOver: (event: DragEvent) => {
                          if (!isDropAccepted(event)) return;
                          event.preventDefault();
                          event.stopPropagation();
                          setDropTarget(r.id);
                      },
                      onDragLeave: () => setDropTarget(undefined),
                      onDrop: (event: DragEvent) => {
                          event.preventDefault();
                          event.stopPropagation();
                          setDropTarget(undefined);
                          const dropped = readDrop(event);
                          if (dropped && !(dropped.kind === "resources" && dropped.ids.includes(r.id)))
                              onDrop(dropped, r.id);
                      },
                  }
                : {}),
        };
    }

    function itemClass(r: Resource) {
        return `${selected.has(r.id) ? "bg-accent/15 ring-1 ring-accent/40" : "hover:bg-gray-100"} ${
            dropTarget === r.id ? "ring-2 ring-accent" : ""
        } ${cutIds.includes(r.id) ? "opacity-50" : ""}`;
    }

    function sortHeader(key: ResourceSortKey, label: string, className = "") {
        const active = sort.key === key;
        return (
            <th
                scope="col"
                aria-sort={active ? (sort.descending ? "descending" : "ascending") : "none"}
                className={`px-3 py-2 font-medium ${className}`}
            >
                <button
                    type="button"
                    onClick={() =>
                        setSort((s) => ({ key, descending: s.key === key ? !s.descending : false }))
                    }
                    className="inline-flex cursor-pointer items-center gap-1 hover:text-gray-900"
                >
                    {label}
                    {active && (sort.descending ? <FiArrowDown aria-hidden /> : <FiArrowUp aria-hidden />)}
                </button>
            </th>
        );
    }

    return (
        <div
            tabIndex={0}
            role="region"
            aria-label={t("resources.folderContents")}
            onKeyDown={handleKeyDown}
            onClick={() => onSelectionChange([])}
            onDragOver={(event) => {
                if (!isDropAccepted(event)) return;
                event.preventDefault();
                setDropTarget(null);
            }}
            onDragLeave={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget as Node)) setDropTarget(undefined);
            }}
            onDrop={(event) => {
                event.preventDefault();
                setDropTarget(undefined);
                const dropped = readDrop(event);
                if (dropped) onDrop(dropped, folderId);
            }}
            className={`min-h-full flex-1 rounded-lg p-2 focus:outline-none ${
                dropTarget === null ? "bg-accent/5 ring-2 ring-accent/40 ring-inset" : ""
            }`}
        >
            {sorted.length === 0 ? (
                <div className="flex h-64 flex-col items-center justify-center gap-1 text-sm text-gray-400">
                    <p>{t("resources.folderEmpty")}</p>
                    <p>{t("resources.dropHint")}</p>
                </div>
            ) : mode === "list" ? (
                <table className="w-full table-fixed text-left text-sm">
                    <thead className="border-b border-gray-200 text-xs text-gray-500">
                        <tr>
                            {sortHeader("name", t("resources.column.name"))}
                            {sortHeader("updatedAt", t("resources.column.modified"), "w-44")}
                            {sortHeader("size", t("resources.column.size"), "w-24")}
                            <th scope="col" className="w-32 px-3 py-2 font-medium">
                                {t("resources.column.kind")}
                            </th>
                            <th scope="col" className="w-10">
                                <span className="sr-only">{t("resources.menu.label")}</span>
                            </th>
                        </tr>
                    </thead>
                    <tbody>
                        {sorted.map((r) => (
                            <tr
                                key={r.id}
                                id={`resource-${r.id}`}
                                aria-selected={selected.has(r.id)}
                                onClick={(event) => select(event, r)}
                                onDoubleClick={() => onOpen(r)}
                                {...itemDrag(r)}
                                className={`group cursor-default ${itemClass(r)}`}
                            >
                                <td className="rounded-l-md px-3 py-1.5">
                                    <span className="flex min-w-0 items-center gap-2">
                                        <ResourceIcon resource={r} className="text-base" />
                                        <span className="truncate" title={r.name}>
                                            {r.name}
                                        </span>
                                    </span>
                                </td>
                                <td className="px-3 py-1.5 text-gray-500">{format.dateTime(r.updatedAt)}</td>
                                <td className="px-3 py-1.5 text-gray-500">
                                    {r.kind === "file" ? formatSize(r.size) : "—"}
                                </td>
                                <td className="truncate px-3 py-1.5 text-gray-500">
                                    {t(kindLabels[fileCategory(r)])}
                                </td>
                                <td className="rounded-r-md pr-1">
                                    <ResourceMenu items={menuItems(r, selection)} />
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            ) : (
                <ul className="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-2">
                    {sorted.map((r) => (
                        <li
                            key={r.id}
                            id={`resource-${r.id}`}
                            aria-selected={selected.has(r.id)}
                            onClick={(event) => select(event, r)}
                            onDoubleClick={() => onOpen(r)}
                            {...itemDrag(r)}
                            className={`group relative flex cursor-default flex-col items-center gap-2 rounded-lg p-3 ${itemClass(r)}`}
                        >
                            <div className="absolute top-1 right-1">
                                <ResourceMenu items={menuItems(r, selection)} />
                            </div>
                            <div className="grid size-20 place-items-center overflow-hidden rounded-md">
                                {fileCategory(r) === "image" ? (
                                    <img
                                        src={resourceContentUrl(r)}
                                        alt=""
                                        loading="lazy"
                                        draggable={false}
                                        className="size-full rounded-md bg-gray-100 object-cover"
                                    />
                                ) : (
                                    <ResourceIcon resource={r} className="text-6xl" />
                                )}
                            </div>
                            <span className="line-clamp-2 w-full text-center text-xs break-all text-gray-800" title={r.name}>
                                {r.name}
                            </span>
                        </li>
                    ))}
                </ul>
            )}
        </div>
    );
}
