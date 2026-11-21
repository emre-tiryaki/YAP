import { useSyncExternalStore } from "react";
import type { AppNotification } from "../types/notification";

// In-app notifications currently on screen, newest last. A module-level
// store so code outside React (the notification poller) can add to it.
const maxToasts = 4;
const listeners = new Set<() => void>();
let toasts: AppNotification[] = [];

function emit() {
    listeners.forEach((listener) => listener());
}

export function pushToasts(items: AppNotification[]) {
    if (items.length === 0) return;
    toasts = [...toasts, ...items].slice(-maxToasts);
    emit();
}

// A plain confirmation in the toast area, e.g. "Link copied".
export function pushInfoToast(title: string, body: string) {
    pushToasts([
        {
            id: `info:${Date.now()}:${body}`,
            kind: "info",
            title,
            body,
            href: null,
        },
    ]);
}

export function dismissToast(id: string) {
    toasts = toasts.filter((toast) => toast.id !== id);
    emit();
}

function subscribe(listener: () => void) {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

export function useToasts(): AppNotification[] {
    return useSyncExternalStore(subscribe, () => toasts);
}
