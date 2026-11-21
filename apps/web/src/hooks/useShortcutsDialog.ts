import { useSyncExternalStore } from "react";

let isOpen = false;
const listeners = new Set<() => void>();

function set(next: boolean) {
    isOpen = next;
    listeners.forEach((listener) => listener());
}

export function openShortcutsDialog() {
    set(true);
}

export function closeShortcutsDialog() {
    set(false);
}

function subscribe(listener: () => void) {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

export function useShortcutsDialog(): boolean {
    return useSyncExternalStore(subscribe, () => isOpen);
}
