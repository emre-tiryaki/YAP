import type { DragEvent } from "react";

// Drag data type for resources dragged inside the app; dragged-in OS files
// come as "Files".
const RESOURCE_DRAG = "application/x-yap-resources";

export function startResourceDrag(event: DragEvent, ids: string[]) {
    event.dataTransfer.setData(RESOURCE_DRAG, JSON.stringify(ids));
    event.dataTransfer.effectAllowed = "copyMove";
}

export function isDropAccepted(event: DragEvent): boolean {
    const types = event.dataTransfer.types;
    return types.includes(RESOURCE_DRAG) || types.includes("Files");
}

export type DroppedItems =
    | { kind: "resources"; ids: string[] }
    | { kind: "files"; files: File[] }
    | null;

export function readDrop(event: DragEvent): DroppedItems {
    const raw = event.dataTransfer.getData(RESOURCE_DRAG);
    if (raw) {
        try {
            const ids: unknown = JSON.parse(raw);
            if (Array.isArray(ids) && ids.every((id) => typeof id === "string"))
                return { kind: "resources", ids };
        } catch {
            return null;
        }
    }
    const files = [...event.dataTransfer.files];
    // Folders dropped from the OS show up as zero-byte entries without a
    // type; folder upload is not supported.
    const plain = files.filter((f) => f.type !== "" || f.size > 0 || f.name.includes("."));
    return plain.length > 0 ? { kind: "files", files: plain } : null;
}
