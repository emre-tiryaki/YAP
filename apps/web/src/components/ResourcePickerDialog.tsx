import { useState, type ReactNode } from "react";
import { FiChevronRight } from "react-icons/fi";
import { LuHardDrive } from "react-icons/lu";
import { useTranslate } from "../hooks/useTranslate";
import type { Resource } from "../types/resource";
import { childrenOf, type ResourceIndex } from "../util/resources";
import { Modal } from "./Modal";
import { ResourceIcon } from "./ResourceIcon";

type ResourcePickerDialogProps = {
    title: string;
    confirmLabel: string;
    index: ResourceIndex;
    // "folder" picks a target folder (the top of the tree included); "any"
    // picks a file or folder to link.
    mode: "folder" | "any";
    // Items that cannot be picked, with everything inside them.
    excludedIds?: Set<string>;
    isPending?: boolean;
    errorMessage?: string | null;
    onPick: (id: string | null) => void;
    onClose: () => void;
};

// A tree to choose a folder (move to…) or any resource (link to a task).
export function ResourcePickerDialog({
    title,
    confirmLabel,
    index,
    mode,
    excludedIds,
    isPending = false,
    errorMessage,
    onPick,
    onClose,
}: ResourcePickerDialogProps) {
    const t = useTranslate();
    // undefined: nothing picked yet; null: the top of the tree.
    const [picked, setPicked] = useState<string | null | undefined>(
        mode === "folder" ? null : undefined,
    );
    const [expanded, setExpanded] = useState<Set<string>>(new Set());

    function toggle(id: string) {
        setExpanded((prev) => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id);
            else next.add(id);
            return next;
        });
    }

    function renderLevel(parentId: string | null, depth: number) {
        const items = childrenOf(index, parentId).filter(
            (r) => (mode === "any" || r.kind === "folder") && !excludedIds?.has(r.id),
        );
        return items.map((r) => (
            <PickerRow
                key={r.id}
                resource={r}
                depth={depth}
                isPicked={picked === r.id}
                isExpanded={expanded.has(r.id)}
                hasChildren={childrenOf(index, r.id).some(
                    (c) => mode === "any" || c.kind === "folder",
                )}
                onPick={() => setPicked(r.id)}
                onToggle={() => toggle(r.id)}
            >
                {expanded.has(r.id) && renderLevel(r.id, depth + 1)}
            </PickerRow>
        ));
    }

    return (
        <Modal isOpen title={title} onClose={onClose}>
            <ul
                role="tree"
                aria-label={title}
                className="max-h-80 overflow-y-auto rounded-md border border-gray-200 p-1 text-sm"
            >
                {mode === "folder" && (
                    <li role="treeitem" aria-selected={picked === null}>
                        <button
                            type="button"
                            onClick={() => setPicked(null)}
                            className={`flex w-full cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-left ${
                                picked === null ? "bg-accent/15 font-medium" : "hover:bg-gray-100"
                            }`}
                        >
                            <LuHardDrive aria-hidden className="text-gray-500" />
                            {t("resources.title")}
                        </button>
                    </li>
                )}
                {renderLevel(null, mode === "folder" ? 1 : 0)}
                {index.byId.size === 0 && (
                    <li className="px-2 py-3 text-gray-400">{t("resources.empty")}</li>
                )}
            </ul>
            {errorMessage && (
                <p role="alert" className="mt-3 text-sm text-red-600">
                    {errorMessage}
                </p>
            )}
            <div className="mt-6 flex justify-end gap-2">
                <button
                    type="button"
                    onClick={onClose}
                    className="cursor-pointer rounded-md border border-gray-300 px-4 py-2 text-sm hover:bg-gray-100"
                >
                    {t("common.cancel")}
                </button>
                <button
                    type="button"
                    disabled={picked === undefined || isPending}
                    onClick={() => picked !== undefined && onPick(picked)}
                    className="cursor-pointer rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:bg-accent/85 disabled:cursor-not-allowed disabled:opacity-50"
                >
                    {confirmLabel}
                </button>
            </div>
        </Modal>
    );
}

type PickerRowProps = {
    resource: Resource;
    depth: number;
    isPicked: boolean;
    isExpanded: boolean;
    hasChildren: boolean;
    onPick: () => void;
    onToggle: () => void;
    children: ReactNode;
};

function PickerRow({
    resource,
    depth,
    isPicked,
    isExpanded,
    hasChildren,
    onPick,
    onToggle,
    children,
}: PickerRowProps) {
    const t = useTranslate();
    return (
        <li role="treeitem" aria-selected={isPicked} aria-expanded={hasChildren ? isExpanded : undefined}>
            <div
                className={`flex items-center rounded ${isPicked ? "bg-accent/15 font-medium" : "hover:bg-gray-100"}`}
                style={{ paddingLeft: depth * 16 }}
            >
                <button
                    type="button"
                    onClick={onToggle}
                    aria-label={t(isExpanded ? "resources.tree.collapse" : "resources.tree.expand")}
                    className={`cursor-pointer rounded p-1 text-gray-400 hover:text-gray-900 ${hasChildren ? "" : "invisible"}`}
                >
                    <FiChevronRight aria-hidden className={`transition-transform ${isExpanded ? "rotate-90" : ""}`} />
                </button>
                <button
                    type="button"
                    onClick={onPick}
                    onDoubleClick={onToggle}
                    className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 py-1.5 pr-2 text-left"
                >
                    <ResourceIcon resource={resource} isOpen={isExpanded} />
                    <span className="truncate">{resource.name}</span>
                </button>
            </div>
            {children && <ul role="group">{children}</ul>}
        </li>
    );
}
