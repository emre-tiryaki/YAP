import { useState } from "react";
import { useNavigate } from "react-router";
import { ApiError, resourceContentUrl } from "../api/client";
import { ConflictDialog, type ConflictChoice } from "../components/ConflictDialog";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { RenameDialog } from "../components/RenameDialog";
import { ResourcePickerDialog } from "../components/ResourcePickerDialog";
import { TaskPickerDialog } from "../components/TaskPickerDialog";
import type { OnConflict, Resource, ResourceKind } from "../types/resource";
import {
    childrenOf,
    isInside,
    MAX_UPLOAD_BYTES,
    nameTaken,
    type ResourceIndex,
} from "../util/resources";
import { setResourceClipboard, useResourceClipboard } from "./useResourceClipboard";
import {
    useCopyResource,
    useCreateFile,
    useCreateFolder,
    useLinkResource,
    useMoveResource,
    useRenameResource,
    useTrashResource,
    useUploadFile,
} from "./useResources";
import { useTranslate, type TranslationKey } from "./useTranslate";

type Prompt = {
    name: string;
    canReplace: boolean;
    hasMore: boolean;
    resolve: (answer: { choice: ConflictChoice; applyToAll: boolean }) => void;
};

type Candidate<T> = {
    item: T;
    name: string;
    // The item already in the target folder, if any.
    existing: Resource | undefined;
    canReplace: boolean;
};

export function errorKey(error: unknown): TranslationKey {
    if (!(error instanceof ApiError)) return "common.error";
    switch (error.message) {
        case "conflict":
            return "resources.error.conflict";
        case "invalid_input":
            return "resources.error.invalid";
        case "too_large":
            return "resources.error.tooLarge";
        case "file_missing":
            return "resources.error.missing";
        case "not_found":
            return "resources.error.notFound";
        default:
            return "common.error";
    }
}

// Default name for a new item that is free in the folder: "New folder",
// "New folder (1)", ...
function freshName(index: ResourceIndex, folderId: string | null, base: string, ext = "") {
    for (let n = 0; ; n++) {
        const name = n === 0 ? `${base}${ext}` : `${base} (${n})${ext}`;
        if (!nameTaken(index, folderId, name)) return name;
    }
}

// File operations of the resources page, with the dialogs they need. Render
// `dialogs` once; call the returned functions from menus, shortcuts and drops.
export function useResourceOperations(index: ResourceIndex) {
    const t = useTranslate();
    const navigate = useNavigate();
    const clipboard = useResourceClipboard();
    const createFolder = useCreateFolder();
    const createFile = useCreateFile();
    const upload = useUploadFile();
    const rename = useRenameResource();
    const move = useMoveResource();
    const copy = useCopyResource();
    const trash = useTrashResource();
    const link = useLinkResource();

    const [notice, setNotice] = useState<string | null>(null);
    const [busy, setBusy] = useState<string | null>(null);
    const [prompt, setPrompt] = useState<Prompt | null>(null);
    const [creating, setCreating] = useState<{ kind: ResourceKind; parentId: string | null } | null>(null);
    const [renaming, setRenaming] = useState<Resource | null>(null);
    const [movingIds, setMovingIds] = useState<string[] | null>(null);
    const [linking, setLinking] = useState<Resource | null>(null);
    const [trashing, setTrashing] = useState<string[] | null>(null);

    function ask(name: string, canReplace: boolean, hasMore: boolean) {
        return new Promise<{ choice: ConflictChoice; applyToAll: boolean }>((resolve) =>
            setPrompt({ name, canReplace, hasMore, resolve }),
        );
    }

    // Asks about each candidate whose name is taken; returns the policy per
    // item, leaving out skipped ones.
    async function resolveConflicts<T>(candidates: Candidate<T>[]) {
        const out: { item: T; onConflict: OnConflict }[] = [];
        let remembered: ConflictChoice | null = null;
        let remaining = candidates.filter((c) => c.existing).length;
        for (const c of candidates) {
            if (!c.existing) {
                out.push({ item: c.item, onConflict: "fail" });
                continue;
            }
            remaining--;
            let choice: ConflictChoice | null = remembered;
            if (!choice) {
                const answer = await ask(c.name, c.canReplace, remaining > 0);
                choice = answer.choice;
                if (answer.applyToAll) remembered = choice;
            }
            if (choice === "skip") continue;
            out.push({
                item: c.item,
                onConflict: choice === "replace" && c.canReplace ? "replace" : "keep_both",
            });
        }
        return out;
    }

    async function runAll<T>(label: string, items: T[], run: (item: T) => Promise<unknown>) {
        const failed: string[] = [];
        setBusy(label);
        let lastError: unknown = null;
        for (const item of items) {
            try {
                await run(item);
            } catch (error) {
                lastError = error;
                failed.push(String(item));
            }
        }
        setBusy(null);
        if (lastError) setNotice(t(errorKey(lastError)));
        return failed.length === 0;
    }

    async function uploadFiles(parentId: string | null, files: File[]) {
        setNotice(null);
        const tooLarge = files.filter((f) => f.size > MAX_UPLOAD_BYTES);
        const ok = files.filter((f) => f.size <= MAX_UPLOAD_BYTES);
        const plan = await resolveConflicts(
            ok.map((file) => {
                const existing = nameTaken(index, parentId, file.name);
                return {
                    item: file,
                    name: file.name,
                    existing,
                    canReplace: existing?.kind === "file",
                };
            }),
        );
        setBusy(t("resources.uploading"));
        let lastError: unknown = null;
        for (const { item, onConflict } of plan) {
            try {
                await upload.mutateAsync({ parentId, file: item, onConflict });
            } catch (error) {
                lastError = error;
            }
        }
        setBusy(null);
        if (tooLarge.length > 0)
            setNotice(
                `${t("resources.error.tooLarge")} ${tooLarge.map((f) => f.name).join(", ")}`,
            );
        else if (lastError) setNotice(t(errorKey(lastError)));
    }

    // Moves (or copies) items into a folder. Copies into their own folder
    // always keep both, like a file manager's "paste here".
    async function transfer(ids: string[], targetId: string | null, mode: "move" | "copy") {
        setNotice(null);
        const items = ids
            .map((id) => index.byId.get(id))
            .filter((r): r is Resource => r !== undefined)
            .filter((r) => !(mode === "move" && r.parentId === targetId));
        if (targetId && items.some((r) => r.kind === "folder" && isInside(index, targetId, r.id))) {
            setNotice(t("resources.error.intoItself"));
            return false;
        }
        const plan = await resolveConflicts(
            items.map((r) => {
                const sameFolder = r.parentId === targetId;
                const existing = sameFolder ? undefined : nameTaken(index, targetId, r.name);
                return {
                    item: r,
                    name: r.name,
                    existing,
                    canReplace: existing?.kind === "file" && r.kind === "file",
                };
            }),
        );
        return runAll(t(mode === "move" ? "resources.moving" : "resources.copying"), plan, ({ item, onConflict }) => {
            const policy = mode === "copy" && item.parentId === targetId ? "keep_both" : onConflict;
            const mutation = mode === "move" ? move : copy;
            return mutation.mutateAsync({ id: item.id, parentId: targetId, onConflict: policy });
        });
    }

    async function paste(targetId: string | null) {
        if (!clipboard) return;
        const ok = await transfer(clipboard.ids, targetId, clipboard.mode === "cut" ? "move" : "copy");
        if (ok && clipboard.mode === "cut") setResourceClipboard(null);
    }

    function download(resource: Resource) {
        const a = document.createElement("a");
        a.href = resourceContentUrl(resource, true);
        a.download = resource.name;
        document.body.append(a);
        a.click();
        a.remove();
    }

    const createDialog = creating && (
        <RenameDialog
            isOpen
            title={t(creating.kind === "folder" ? "resources.newFolder" : "resources.newFile")}
            label={t("resources.name")}
            initialName={
                creating.kind === "folder"
                    ? freshName(index, creating.parentId, t("resources.newFolder.default"))
                    : freshName(index, creating.parentId, t("resources.newFile.default"), ".md")
            }
            isPending={createFolder.isPending || createFile.isPending}
            errorMessage={
                createFolder.error || createFile.error
                    ? t(errorKey(createFolder.error ?? createFile.error))
                    : null
            }
            onSubmit={(name) => {
                const mutation = creating.kind === "folder" ? createFolder : createFile;
                mutation.mutate(
                    { parentId: creating.parentId, name },
                    {
                        onSuccess: (created) => {
                            setCreating(null);
                            if (created.kind === "file") void navigate(`/resources/${created.id}`);
                        },
                    },
                );
            }}
            onClose={() => {
                setCreating(null);
                createFolder.reset();
                createFile.reset();
            }}
        />
    );

    const renameDialog = renaming && (
        <RenameDialog
            isOpen
            title={t("resources.rename")}
            label={t("resources.name")}
            initialName={renaming.name}
            isPending={rename.isPending}
            errorMessage={rename.error ? t(errorKey(rename.error)) : null}
            onSubmit={(name) =>
                rename.mutate(
                    { id: renaming.id, name },
                    { onSuccess: () => setRenaming(null) },
                )
            }
            onClose={() => {
                setRenaming(null);
                rename.reset();
            }}
        />
    );

    const excluded = new Set(
        (movingIds ?? []).filter((id) => index.byId.get(id)?.kind === "folder"),
    );
    const moveDialog = movingIds && (
        <ResourcePickerDialog
            title={t("resources.moveTo")}
            confirmLabel={t("resources.moveHere")}
            index={index}
            mode="folder"
            excludedIds={excluded}
            onPick={(target) => {
                const ids = movingIds;
                setMovingIds(null);
                void transfer(ids, target, "move");
            }}
            onClose={() => setMovingIds(null)}
        />
    );

    const linkDialog = linking && (
        <TaskPickerDialog
            title={`${t("resources.linkToTask")}: ${linking.name}`}
            isPending={link.isPending}
            errorMessage={link.error ? t(errorKey(link.error)) : null}
            onPick={(taskId) =>
                link.mutate(
                    { taskId, resourceId: linking.id },
                    { onSuccess: () => setLinking(null) },
                )
            }
            onClose={() => {
                setLinking(null);
                link.reset();
            }}
        />
    );

    const trashDialog = trashing && (
        <ConfirmDialog
            isOpen
            title={t("resources.trash.confirmTitle")}
            message={
                trashing.length === 1
                    ? t("resources.trash.confirmOne").replace("{name}", index.byId.get(trashing[0])?.name ?? "")
                    : t("resources.trash.confirmMany").replace("{count}", String(trashing.length))
            }
            confirmLabel={t("resources.moveToTrash")}
            isPending={trash.isPending}
            onConfirm={() => {
                const ids = trashing;
                setTrashing(null);
                void runAll(t("resources.trashing"), ids, (id) => trash.mutateAsync(id));
            }}
            onClose={() => setTrashing(null)}
        />
    );

    const conflictDialog = prompt && (
        <ConflictDialog
            key={prompt.name}
            name={prompt.name}
            canReplace={prompt.canReplace}
            hasMore={prompt.hasMore}
            onChoose={(choice, applyToAll) => {
                setPrompt(null);
                prompt.resolve({ choice, applyToAll });
            }}
        />
    );

    return {
        notice,
        busy,
        clearNotice: () => setNotice(null),
        clipboard,
        startCreate: (kind: ResourceKind, parentId: string | null) => setCreating({ kind, parentId }),
        startRename: (resource: Resource) => setRenaming(resource),
        startMove: (ids: string[]) => setMovingIds(ids),
        startLink: (resource: Resource) => setLinking(resource),
        // Asks before trashing a folder with contents; plain items go at once.
        trash: (ids: string[]) => {
            if (ids.some((id) => childrenOf(index, id).length > 0)) setTrashing(ids);
            else void runAll(t("resources.trashing"), ids, (id) => trash.mutateAsync(id));
        },
        cut: (ids: string[]) => setResourceClipboard({ mode: "cut", ids }),
        copy: (ids: string[]) => setResourceClipboard({ mode: "copy", ids }),
        paste,
        move: (ids: string[], targetId: string | null) => transfer(ids, targetId, "move"),
        uploadFiles,
        download,
        dialogs: (
            <>
                {createDialog}
                {renameDialog}
                {moveDialog}
                {linkDialog}
                {trashDialog}
                {conflictDialog}
            </>
        ),
    };
}

export type ResourceOperations = ReturnType<typeof useResourceOperations>;
