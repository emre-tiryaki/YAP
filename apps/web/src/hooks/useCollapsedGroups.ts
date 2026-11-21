import { useSyncExternalStore } from "react";

// Which groups (columns or list groups) of a plan are collapsed. A per-device
// UI state, kept in localStorage under yap.plan.<planId>.<kind>.
const listeners = new Set<() => void>();
const caches = new Map<string, string[]>();

function storageKey(planId: string, kind: string) {
    return `yap.plan.${planId}.${kind}`;
}

function read(key: string): string[] {
    const cached = caches.get(key);
    if (cached) return cached;
    let value: string[] = [];
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(key) ?? "[]");
        if (Array.isArray(parsed))
            value = parsed.filter((id): id is string => typeof id === "string");
    } catch {
        // Corrupt value: nothing collapsed.
    }
    caches.set(key, value);
    return value;
}

export function toggleCollapsedGroup(
    planId: string,
    kind: "collapsed" | "listCollapsed",
    groupId: string,
) {
    const key = storageKey(planId, kind);
    const current = read(key);
    const next = current.includes(groupId)
        ? current.filter((id) => id !== groupId)
        : [...current, groupId];
    caches.set(key, next);
    localStorage.setItem(key, JSON.stringify(next));
    listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void) {
    listeners.add(listener);
    function handleStorage() {
        caches.clear();
        listener();
    }
    window.addEventListener("storage", handleStorage);
    return () => {
        listeners.delete(listener);
        window.removeEventListener("storage", handleStorage);
    };
}

export function useCollapsedGroups(
    planId: string,
    kind: "collapsed" | "listCollapsed",
): string[] {
    const key = storageKey(planId, kind);
    return useSyncExternalStore(subscribe, () => read(key));
}
