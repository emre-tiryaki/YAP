import { useRef, useState, type KeyboardEvent, type PointerEvent, type ReactNode } from "react";
import { FiClipboard, FiDownload, FiFilePlus, FiFolderPlus, FiGithub, FiGrid, FiList, FiRotateCcw, FiTrash2, FiUpload } from "react-icons/fi";
import { useNavigate, useParams } from "react-router";
import { AddRepositoryDialog } from "../components/AddRepositoryDialog";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { FilePreview } from "../components/FilePreview";
import { FolderView } from "../components/FolderView";
import { InlineError } from "../components/InlineError";
import { LinkedTaskList } from "../components/LinkedTaskList";
import { ResourceActivity } from "../components/ResourceActivity";
import { ResourceIcon } from "../components/ResourceIcon";
import { ResourceMenu } from "../components/ResourceMenu";
import { RepositoryPane } from "../components/RepositoryPane";
import { ResourceTree } from "../components/ResourceTree";
import { Tabs } from "../components/Tabs";
import { PageHeader } from "../components/PageHeader";
import { useDateFormat } from "../hooks/useDateFormat";
import { useResourceMenuItems } from "../hooks/useResourceMenuItems";
import { errorKey, useResourceOperations, type ResourceOperations } from "../hooks/useResourceOperations";
import { useEmptyTrash, usePurgeResource, useResource, useResources, useRestoreResource, useTrash } from "../hooks/useResources";
import { useTranslate } from "../hooks/useTranslate";
import type { BreadcrumbItem } from "../types/breadcrumb";
import type { Resource, ResourceViewMode } from "../types/resource";
import type { DroppedItems } from "../util/resourceDrag";
import { childrenOf, formatSize, indexResources, pathTo, type ResourceIndex } from "../util/resources";

const WIDTH_KEY = "yap.resources.treeWidth";
const VIEW_KEY = "yap.resources.view";
const MIN_WIDTH = 180;
const MAX_WIDTH = 600;

function clampWidth(width: number) {
    return Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, width));
}

type Tab = "content" | "activity" | "tasks";

export default function ResourcesPage({ view = "browse" }: { view?: "browse" | "trash" }) {
    const t = useTranslate();
    const navigate = useNavigate();
    const { resourceId = "" } = useParams();
    const resources = useResources();
    const trash = useTrash();
    const index = indexResources(resources.data ?? []);
    const ops = useResourceOperations(index);
    const menuItems = useResourceMenuItems(ops);

    const live = resourceId ? index.byId.get(resourceId) : undefined;
    // An id outside the live tree may be in the trash (or gone).
    const fallback = useResource(resourceId, resources.isSuccess && resourceId !== "" && !live);
    const current: Resource | undefined = live ?? fallback.data;

    const [width, setWidth] = useState(() => clampWidth(Number(localStorage.getItem(WIDTH_KEY)) || 280));
    const [mode, setMode] = useState<ResourceViewMode>(() =>
        localStorage.getItem(VIEW_KEY) === "grid" ? "grid" : "list",
    );

    function changeMode(next: ResourceViewMode) {
        setMode(next);
        localStorage.setItem(VIEW_KEY, next);
    }

    function handleDrop(items: NonNullable<DroppedItems>, targetId: string | null) {
        if (items.kind === "files") void ops.uploadFiles(targetId, items.files);
        else void ops.move(items.ids, targetId);
    }

    const folderPath = pathTo(index, current?.trashedAt ? null : (current?.id ?? null));
    const breadcrumb: BreadcrumbItem[] = [
        { label: t("resources.title"), to: "/resources" },
        ...(view === "trash" ? [{ label: t("resources.trash"), to: "/resources/trash" }] : []),
        ...folderPath.map((r) => ({ label: r.name, to: `/resources/${r.id}` })),
        ...(current?.trashedAt ? [{ label: current.name, to: `/resources/${current.id}` }] : []),
    ];

    let body: ReactNode;
    if (resources.isLoading) {
        body = <p className="p-6 text-sm text-gray-400">{t("common.loading")}</p>;
    } else if (resources.isError) {
        body = (
            <div className="p-6">
                <InlineError message={t("resources.loadError")} onRetry={() => void resources.refetch()} />
            </div>
        );
    } else if (view === "trash") {
        body = <TrashPane items={trash.data ?? []} isLoading={trash.isLoading} />;
    } else if (resourceId && !current) {
        body = fallback.isLoading ? (
            <p className="p-6 text-sm text-gray-400">{t("common.loading")}</p>
        ) : (
            <p className="p-6 text-sm text-gray-500">{t("resources.error.notFound")}</p>
        );
    } else if (current?.isRepository && !current.trashedAt) {
        body = (
            <RepositoryPane
                key={current.id}
                resource={current}
                menu={menuItems(current)}
                onLink={() => ops.startLink(current)}
            />
        );
    } else if (current?.kind === "file" || current?.trashedAt) {
        body = <ItemPane key={current.id} resource={current} ops={ops} menu={menuItems(current)} />;
    } else {
        body = (
            <FolderPane
                key={current?.id ?? "top"}
                folder={current ?? null}
                index={index}
                ops={ops}
                mode={mode}
                onModeChange={changeMode}
                menuItems={menuItems}
                onDrop={handleDrop}
                onOpen={(r) => void navigate(`/resources/${r.id}`)}
            />
        );
    }

    return (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <PageHeader breadcrumb={breadcrumb} />
            <div className="flex min-h-0 flex-1">
                <aside style={{ width }} className="flex shrink-0 flex-col border-r border-gray-200 bg-gray-50">
                    <ResourceTree
                        index={index}
                        selectedId={view === "trash" ? undefined : current?.trashedAt ? undefined : (current?.id ?? null)}
                        menuItems={(r) => menuItems(r)}
                        onDrop={handleDrop}
                        onRename={ops.startRename}
                        onTrash={(r) => ops.trash([r.id])}
                        trashCount={trash.data?.length ?? 0}
                    />
                </aside>
                <ResizeHandle width={width} onChange={setWidth} />
                <main className="flex min-w-0 flex-1 flex-col">
                    {(ops.notice || ops.busy) && (
                        <div
                            role={ops.notice ? "alert" : "status"}
                            className={`flex items-center gap-3 border-b px-4 py-2 text-sm ${
                                ops.notice ? "border-red-200 bg-red-50 text-red-700" : "border-gray-200 bg-gray-50 text-gray-600"
                            }`}
                        >
                            <span className="flex-1">{ops.notice ?? ops.busy}</span>
                            {ops.notice && (
                                <button type="button" onClick={ops.clearNotice} className="cursor-pointer text-xs underline">
                                    {t("common.close")}
                                </button>
                            )}
                        </div>
                    )}
                    {body}
                </main>
            </div>
            {ops.dialogs}
        </div>
    );
}

// The draggable border between the tree and the content. Arrow keys resize
// it too; the width is remembered.
function ResizeHandle({ width, onChange }: { width: number; onChange: (width: number) => void }) {
    const t = useTranslate();
    const start = useRef<{ x: number; width: number } | null>(null);

    function commit(next: number) {
        const clamped = clampWidth(next);
        onChange(clamped);
        localStorage.setItem(WIDTH_KEY, String(clamped));
    }

    return (
        <div
            role="separator"
            aria-orientation="vertical"
            aria-label={t("resources.tree.resize")}
            aria-valuemin={MIN_WIDTH}
            aria-valuemax={MAX_WIDTH}
            aria-valuenow={width}
            tabIndex={0}
            onPointerDown={(event: PointerEvent<HTMLDivElement>) => {
                event.currentTarget.setPointerCapture(event.pointerId);
                start.current = { x: event.clientX, width };
            }}
            onPointerMove={(event: PointerEvent<HTMLDivElement>) => {
                if (!start.current) return;
                onChange(clampWidth(start.current.width + event.clientX - start.current.x));
            }}
            onPointerUp={(event: PointerEvent<HTMLDivElement>) => {
                if (!start.current) return;
                commit(start.current.width + event.clientX - start.current.x);
                start.current = null;
            }}
            onDoubleClick={() => commit(280)}
            onKeyDown={(event: KeyboardEvent<HTMLDivElement>) => {
                if (event.key === "ArrowLeft") commit(width - 16);
                else if (event.key === "ArrowRight") commit(width + 16);
                else return;
                event.preventDefault();
            }}
            className="w-1 shrink-0 cursor-col-resize bg-transparent transition-colors hover:bg-accent/40 focus:bg-accent/40 focus:outline-none active:bg-accent/60"
        />
    );
}

type FolderPaneProps = {
    folder: Resource | null;
    index: ResourceIndex;
    ops: ResourceOperations;
    mode: ResourceViewMode;
    onModeChange: (mode: ResourceViewMode) => void;
    menuItems: ReturnType<typeof useResourceMenuItems>;
    onDrop: (items: NonNullable<DroppedItems>, targetId: string | null) => void;
    onOpen: (resource: Resource) => void;
};

function FolderPane({ folder, index, ops, mode, onModeChange, menuItems, onDrop, onOpen }: FolderPaneProps) {
    const t = useTranslate();
    const [tab, setTab] = useState<Tab>("content");
    const [selection, setSelection] = useState<string[]>([]);
    const [isAddingRepo, setIsAddingRepo] = useState(false);
    const fileInput = useRef<HTMLInputElement>(null);
    const folderId = folder?.id ?? null;
    const items = childrenOf(index, folderId);
    // Selected ids that no longer exist (moved, trashed) drop out.
    const liveSelection = selection.filter((id) => index.byId.get(id)?.parentId === folderId);

    const tabs: { id: Tab; label: string }[] = [
        { id: "content", label: t("resources.tab.content") },
        { id: "activity", label: t("resources.tab.activity") },
        ...(folder ? [{ id: "tasks" as const, label: t("resources.tab.tasks") }] : []),
    ];

    const toolButton =
        "flex cursor-pointer items-center gap-1.5 rounded-md px-2.5 py-1.5 text-sm text-gray-700 hover:bg-gray-100 disabled:cursor-default disabled:opacity-40";

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <header className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-gray-200 px-4 pt-3">
                <h1 className="flex min-w-0 items-center gap-2 pb-2 text-lg font-semibold text-gray-900">
                    {folder && <ResourceIcon resource={folder} isOpen className="text-xl" />}
                    <span className="truncate">{folder?.name ?? t("resources.title")}</span>
                    {folder && <ResourceMenu items={menuItems(folder)} isSubtle={false} />}
                </h1>
                <span className="flex-1" />
                <Tabs value={tab} onChange={setTab} tabs={tabs} />
            </header>
            {tab === "content" && (
                <>
                    <div className="flex flex-wrap items-center gap-1 border-b border-gray-100 px-3 py-1.5">
                        <button type="button" className={toolButton} onClick={() => ops.startCreate("folder", folderId)}>
                            <FiFolderPlus aria-hidden />
                            {t("resources.newFolder")}
                        </button>
                        <button type="button" className={toolButton} onClick={() => ops.startCreate("file", folderId)}>
                            <FiFilePlus aria-hidden />
                            {t("resources.newFile")}
                        </button>
                        <button type="button" className={toolButton} onClick={() => fileInput.current?.click()}>
                            <FiUpload aria-hidden />
                            {t("resources.upload")}
                        </button>
                        <input
                            ref={fileInput}
                            type="file"
                            multiple
                            hidden
                            onChange={(event) => {
                                const files = [...(event.target.files ?? [])];
                                event.target.value = "";
                                if (files.length > 0) void ops.uploadFiles(folderId, files);
                            }}
                        />
                        <button type="button" className={toolButton} disabled={!ops.clipboard} onClick={() => void ops.paste(folderId)}>
                            <FiClipboard aria-hidden />
                            {t("resources.paste")}
                        </button>
                        <button type="button" className={toolButton} onClick={() => setIsAddingRepo(true)}>
                            <FiGithub aria-hidden />
                            {t("resources.addRepository")}
                        </button>
                        <AddRepositoryDialog
                            isOpen={isAddingRepo}
                            parentId={folderId}
                            onClose={() => setIsAddingRepo(false)}
                            onAdded={(resource) => {
                                setIsAddingRepo(false);
                                onOpen(resource);
                            }}
                        />
                        <span className="flex-1" />
                        {liveSelection.length > 1 && (
                            <span className="px-2 text-xs text-gray-500">
                                {t("resources.selected").replace("{count}", String(liveSelection.length))}
                            </span>
                        )}
                        <div role="radiogroup" aria-label={t("resources.view.label")} className="flex rounded-md border border-gray-300 p-0.5">
                            {(
                                [
                                    ["list", FiList, "resources.view.list"],
                                    ["grid", FiGrid, "resources.view.grid"],
                                ] as const
                            ).map(([value, Icon, label]) => (
                                <button
                                    key={value}
                                    type="button"
                                    role="radio"
                                    aria-checked={mode === value}
                                    aria-label={t(label)}
                                    title={t(label)}
                                    onClick={() => onModeChange(value)}
                                    className={`cursor-pointer rounded px-2 py-1 ${mode === value ? "bg-accent text-accent-fg" : "text-gray-600 hover:bg-gray-100"}`}
                                >
                                    <Icon aria-hidden />
                                </button>
                            ))}
                        </div>
                    </div>
                    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto p-2">
                        <FolderView
                            folderId={folderId}
                            items={items}
                            mode={mode}
                            selection={liveSelection}
                            onSelectionChange={setSelection}
                            onOpen={onOpen}
                            menuItems={menuItems}
                            onDrop={onDrop}
                            onRename={ops.startRename}
                            onTrash={ops.trash}
                            onCut={ops.cut}
                            onCopy={ops.copy}
                            onPaste={() => void ops.paste(folderId)}
                            cutIds={ops.clipboard?.mode === "cut" ? ops.clipboard.ids : []}
                        />
                    </div>
                </>
            )}
            {tab === "activity" && (
                <div className="flex-1 overflow-y-auto p-6">
                    <ResourceActivity scope={folder ? { kind: "resource", id: folder.id } : { kind: "workspace" }} />
                </div>
            )}
            {tab === "tasks" && folder && (
                <div className="flex-1 overflow-y-auto p-6">
                    <LinkedTaskList resourceId={folder.id} onLink={() => ops.startLink(folder)} />
                </div>
            )}
        </div>
    );
}

type ItemPaneProps = {
    resource: Resource;
    ops: ResourceOperations;
    menu: ReturnType<ReturnType<typeof useResourceMenuItems>>;
};

// A file (or any trashed item): its preview, activity and linked tasks.
function ItemPane({ resource, ops, menu }: ItemPaneProps) {
    const t = useTranslate();
    const format = useDateFormat();
    const restore = useRestoreResource();
    const [tab, setTab] = useState<Tab>("content");
    const isTrashed = resource.trashedAt !== null;
    const tabs: { id: Tab; label: string }[] = [
        { id: "content", label: t(resource.kind === "file" ? "resources.tab.preview" : "resources.tab.content") },
        { id: "activity", label: t("resources.tab.activity") },
        { id: "tasks", label: t("resources.tab.tasks") },
    ];

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <header className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-gray-200 px-4 pt-3">
                <div className="min-w-0 pb-2">
                    <h1 className="flex min-w-0 items-center gap-2 text-lg font-semibold text-gray-900">
                        <ResourceIcon resource={resource} className="text-xl" />
                        <span className="truncate">{resource.name}</span>
                        {!isTrashed && <ResourceMenu items={menu} isSubtle={false} />}
                    </h1>
                    <p className="text-xs text-gray-500">
                        {resource.kind === "file" && `${formatSize(resource.size)} · `}
                        {t("resources.column.modified")}: {format.dateTime(resource.updatedAt)}
                    </p>
                </div>
                <span className="flex-1" />
                {resource.kind === "file" && (
                    <button
                        type="button"
                        onClick={() => ops.download(resource)}
                        className="mb-2 flex cursor-pointer items-center gap-1.5 rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-100"
                    >
                        <FiDownload aria-hidden />
                        {t("resources.download")}
                    </button>
                )}
                <Tabs value={tab} onChange={setTab} tabs={tabs} />
            </header>
            {isTrashed && (
                <div role="status" className="flex items-center gap-3 border-b border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-800">
                    <span className="flex-1">
                        {t(resource.trashedRootId === resource.id ? "resources.trashedNotice" : "resources.trashedInsideNotice")}
                    </span>
                    {resource.trashedRootId === resource.id && (
                        <button
                            type="button"
                            disabled={restore.isPending}
                            onClick={() => restore.mutate(resource.id)}
                            className="cursor-pointer rounded-md border border-amber-300 bg-white px-3 py-1 hover:bg-amber-100"
                        >
                            {t("resources.restore")}
                        </button>
                    )}
                </div>
            )}
            {tab === "content" &&
                (resource.kind === "file" ? (
                    <FilePreview resource={resource} onDownload={() => ops.download(resource)} />
                ) : (
                    <p className="p-6 text-sm text-gray-500">{t("resources.trashedFolder")}</p>
                ))}
            {tab === "activity" && (
                <div className="flex-1 overflow-y-auto p-6">
                    <ResourceActivity scope={{ kind: "resource", id: resource.id }} />
                </div>
            )}
            {tab === "tasks" && (
                <div className="flex-1 overflow-y-auto p-6">
                    <LinkedTaskList resourceId={resource.id} onLink={isTrashed ? undefined : () => ops.startLink(resource)} />
                </div>
            )}
        </div>
    );
}

function TrashPane({ items, isLoading }: { items: Resource[]; isLoading: boolean }) {
    const t = useTranslate();
    const format = useDateFormat();
    const restore = useRestoreResource();
    const purge = usePurgeResource();
    const empty = useEmptyTrash();
    const [confirm, setConfirm] = useState<Resource | "all" | null>(null);
    const error = restore.error ?? purge.error ?? empty.error;

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <header className="flex items-center gap-4 border-b border-gray-200 px-4 py-3">
                <h1 className="flex items-center gap-2 text-lg font-semibold text-gray-900">
                    <FiTrash2 aria-hidden className="text-gray-500" />
                    {t("resources.trash")}
                </h1>
                <span className="flex-1" />
                <button
                    type="button"
                    disabled={items.length === 0}
                    onClick={() => setConfirm("all")}
                    className="cursor-pointer rounded-md border border-red-200 px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 disabled:cursor-default disabled:opacity-40"
                >
                    {t("resources.trash.empty")}
                </button>
            </header>
            {error && (
                <p role="alert" className="border-b border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">
                    {t(errorKey(error))}
                </p>
            )}
            <div className="flex-1 overflow-y-auto p-4">
                {isLoading && <p className="text-sm text-gray-400">{t("common.loading")}</p>}
                {!isLoading && items.length === 0 && <p className="text-sm text-gray-400">{t("resources.trash.nothing")}</p>}
                {items.length > 0 && (
                    <table className="w-full table-fixed text-left text-sm">
                        <thead className="border-b border-gray-200 text-xs text-gray-500">
                            <tr>
                                <th scope="col" className="px-3 py-2 font-medium">{t("resources.column.name")}</th>
                                <th scope="col" className="w-44 px-3 py-2 font-medium">{t("resources.column.trashed")}</th>
                                <th scope="col" className="w-64 px-3 py-2"><span className="sr-only">{t("resources.menu.label")}</span></th>
                            </tr>
                        </thead>
                        <tbody>
                            {items.map((r) => (
                                <tr key={r.id} className="hover:bg-gray-50">
                                    <td className="px-3 py-1.5">
                                        <span className="flex min-w-0 items-center gap-2">
                                            <ResourceIcon resource={r} className="text-base" />
                                            <span className="truncate" title={r.name}>{r.name}</span>
                                        </span>
                                    </td>
                                    <td className="px-3 py-1.5 text-gray-500">{r.trashedAt && format.dateTime(r.trashedAt)}</td>
                                    <td className="px-3 py-1.5 text-right">
                                        <button
                                            type="button"
                                            onClick={() => restore.mutate(r.id)}
                                            className="mr-2 inline-flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-gray-700 hover:bg-gray-100"
                                        >
                                            <FiRotateCcw aria-hidden />
                                            {t("resources.restore")}
                                        </button>
                                        <button
                                            type="button"
                                            onClick={() => setConfirm(r)}
                                            className="inline-flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-red-600 hover:bg-red-50"
                                        >
                                            <FiTrash2 aria-hidden />
                                            {t("resources.purge")}
                                        </button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>
            <ConfirmDialog
                isOpen={confirm !== null}
                title={t(confirm === "all" ? "resources.trash.empty" : "resources.purge")}
                message={
                    confirm === "all"
                        ? t("resources.trash.emptyConfirm")
                        : t("resources.purgeConfirm").replace("{name}", confirm?.name ?? "")
                }
                confirmLabel={t(confirm === "all" ? "resources.trash.empty" : "resources.purge")}
                isPending={purge.isPending || empty.isPending}
                onConfirm={() => {
                    const target = confirm;
                    setConfirm(null);
                    if (target === "all") empty.mutate();
                    else if (target) purge.mutate(target.id);
                }}
                onClose={() => setConfirm(null)}
            />
        </div>
    );
}
