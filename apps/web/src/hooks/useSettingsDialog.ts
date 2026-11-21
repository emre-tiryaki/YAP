import { useSyncExternalStore } from "react";
import type { SettingsSection } from "../types/settings";

// Which settings section is open, or null. One SettingsModal in the app shell
// reads it, so the profile menu, the palette and the shortcuts can open it.
let section: SettingsSection | null = null;
const listeners = new Set<() => void>();

function set(next: SettingsSection | null) {
    section = next;
    listeners.forEach((listener) => listener());
}

export function openSettings(next: SettingsSection = "general") {
    set(next);
}

export function closeSettings() {
    set(null);
}

function subscribe(listener: () => void) {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

export function useSettingsDialog(): SettingsSection | null {
    return useSyncExternalStore(subscribe, () => section);
}
