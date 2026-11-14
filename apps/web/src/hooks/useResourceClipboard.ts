import { useSyncExternalStore } from "react";

export type ResourceClipboard = {
    mode: "cut" | "copy";
    ids: string[];
} | null;

// In memory only: like a file manager, cut/copy lasts until the app reloads.
let clipboard: ResourceClipboard = null;
const listeners = new Set<() => void>();

export function setResourceClipboard(next: ResourceClipboard) {
    clipboard = next;
    listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void) {
    listeners.add(listener);
    return () => listeners.delete(listener);
}

export function useResourceClipboard(): ResourceClipboard {
    return useSyncExternalStore(subscribe, () => clipboard);
}
