import type { Resource } from "../types/resource";

// Names sort like a file manager: case-insensitive, with numbers in order
// ("file 2" before "file 10").
const collator = new Intl.Collator(undefined, {
    numeric: true,
    sensitivity: "base",
});

export type ResourceSortKey = "name" | "updatedAt" | "size";

// Folders always come first; then the chosen column, ties broken by name.
export function compareResources(
    a: Resource,
    b: Resource,
    key: ResourceSortKey = "name",
    descending = false,
): number {
    if (a.kind !== b.kind) return a.kind === "folder" ? -1 : 1;
    let result = 0;
    if (key === "updatedAt")
        result = a.updatedAt.getTime() - b.updatedAt.getTime();
    if (key === "size") result = a.size - b.size;
    if (result === 0) result = collator.compare(a.name, b.name);
    return descending ? -result : result;
}

export type ResourceIndex = {
    byId: Map<string, Resource>;
    // Children by parent id; "" is the top of the tree. Sorted by name.
    children: Map<string, Resource[]>;
};

export function indexResources(resources: Resource[]): ResourceIndex {
    const byId = new Map(resources.map((r) => [r.id, r]));
    const children = new Map<string, Resource[]>();
    for (const r of resources) {
        const key = r.parentId ?? "";
        const list = children.get(key);
        if (list) list.push(r);
        else children.set(key, [r]);
    }
    for (const list of children.values()) list.sort((a, b) => compareResources(a, b));
    return { byId, children };
}

export function childrenOf(index: ResourceIndex, folderId: string | null) {
    return index.children.get(folderId ?? "") ?? [];
}

// The folders from the top down to (and including) id.
export function pathTo(index: ResourceIndex, id: string | null): Resource[] {
    const path: Resource[] = [];
    let current = id ? index.byId.get(id) : undefined;
    while (current) {
        path.unshift(current);
        current = current.parentId ? index.byId.get(current.parentId) : undefined;
    }
    return path;
}

// Whether id is folderId or sits somewhere inside it.
export function isInside(
    index: ResourceIndex,
    id: string | null,
    folderId: string,
): boolean {
    return pathTo(index, id).some((r) => r.id === folderId);
}

export function nameTaken(
    index: ResourceIndex,
    folderId: string | null,
    name: string,
    exceptId?: string,
): Resource | undefined {
    const key = name.trim().toLowerCase();
    return childrenOf(index, folderId).find(
        (r) => r.id !== exceptId && r.name.toLowerCase() === key,
    );
}

const units = ["B", "KB", "MB", "GB"];

export function formatSize(bytes: number): string {
    let value = bytes;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
        value /= 1024;
        unit++;
    }
    const digits = unit === 0 || value >= 10 ? 0 : 1;
    return `${value.toFixed(digits)} ${units[unit]}`;
}

export type FileCategory =
    | "folder"
    | "image"
    | "pdf"
    | "markdown"
    | "text"
    | "audio"
    | "video"
    | "other";

function extension(name: string): string {
    const dot = name.lastIndexOf(".");
    return dot > 0 ? name.slice(dot + 1).toLowerCase() : "";
}

// Decides how a resource is previewed.
// The fields that decide how a resource is shown.
export type ResourceLook = Pick<Resource, "kind" | "name" | "mimeType" | "isText">;

export function fileCategory(resource: ResourceLook): FileCategory {
    if (resource.kind === "folder") return "folder";
    const mime = resource.mimeType;
    const ext = extension(resource.name);
    if (ext === "md" || ext === "markdown" || mime === "text/markdown")
        return resource.isText ? "markdown" : "other";
    if (mime.startsWith("image/")) return "image";
    if (mime === "application/pdf") return "pdf";
    if (mime.startsWith("audio/")) return "audio";
    if (mime.startsWith("video/")) return "video";
    if (resource.isText) return "text";
    return "other";
}

// Upload limit, mirrored from the backend.
export const MAX_UPLOAD_BYTES = 100 * 1024 * 1024;
