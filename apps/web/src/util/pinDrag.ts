import type { DragEvent } from "react";
import { isPin } from "../hooks/usePins";
import type { Pin } from "../types/pin";
import type { Resource } from "../types/resource";

// Anything that can be pinned carries its Pin under this type while dragged,
// so dropping it on the sidebar's Pinned section pins it.
export const PIN_DRAG = "application/x-yap-pin";

export function setPinDragData(event: DragEvent, pin: Pin) {
    event.dataTransfer.setData(PIN_DRAG, JSON.stringify(pin));
    // Keep any effect another drag source already allowed (e.g. resource moves).
    if (event.dataTransfer.effectAllowed === "uninitialized")
        event.dataTransfer.effectAllowed = "copyLink";
}

// Only the type list is readable during dragover; the data itself only on drop.
export function hasPinDrag(event: { dataTransfer: DataTransfer | null }): boolean {
    return event.dataTransfer?.types.includes(PIN_DRAG) ?? false;
}

export function readPinDrag(event: DragEvent): Pin | null {
    const raw = event.dataTransfer.getData(PIN_DRAG);
    if (!raw) return null;
    try {
        const value: unknown = JSON.parse(raw);
        return isPin(value) ? value : null;
    } catch {
        // Not ours or malformed: ignore the drop.
        return null;
    }
}

export function resourcePin(resource: Resource): Pin {
    return { kind: "resource", id: resource.id, label: resource.name, href: `/resources/${resource.id}` };
}
