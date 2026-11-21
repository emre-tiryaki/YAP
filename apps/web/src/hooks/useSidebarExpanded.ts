import { useSyncExternalStore } from "react";

// Which projects are expanded in the sidebar. A per-device UI state, so it
// lives in localStorage like pins and preferences.
const STORAGE_KEY = "yap.sidebar.expanded";

const listeners = new Set<() => void>();
// Cached so useSyncExternalStore gets the same array until something changes.
let cache: string[] | null = null;

function read(): string[] {
    if (cache) return cache;
    try {
        const parsed: unknown = JSON.parse(
            localStorage.getItem(STORAGE_KEY) ?? "[]",
        );
        cache = Array.isArray(parsed)
            ? parsed.filter((id): id is string => typeof id === "string")
            : [];
    } catch {
        cache = [];
    }
    return cache;
}

function subscribe(listener: () => void) {
    listeners.add(listener);
    function handleStorage(event: StorageEvent) {
        if (event.key !== STORAGE_KEY) return;
        cache = null;
        listener();
    }
    window.addEventListener("storage", handleStorage);
    return () => {
        listeners.delete(listener);
        window.removeEventListener("storage", handleStorage);
    };
}

export function toggleSidebarExpanded(projectId: string) {
    const current = read();
    cache = current.includes(projectId)
        ? current.filter((id) => id !== projectId)
        : [...current, projectId];
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cache));
    listeners.forEach((listener) => listener());
}

export function useSidebarExpanded(): string[] {
    return useSyncExternalStore(subscribe, read);
}
