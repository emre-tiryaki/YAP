import { useSyncExternalStore } from "react";

export type PalettePage = "root" | "status" | "priority" | "labels";

// What the palette acts on besides the workspace: set by the task peek (or
// task page) while it is mounted.
export type PaletteContext = {
    kind: "task";
    taskId: string;
    projectId: string;
    planId: string;
    title: string;
};

type PaletteState = {
    isOpen: boolean;
    page: PalettePage;
    context: PaletteContext | null;
};

// A tiny external store, like usePreferences, so any component (the header
// search button, the shortcuts, the peek) can open the palette.
let state: PaletteState = { isOpen: false, page: "root", context: null };
const listeners = new Set<() => void>();

function set(next: Partial<PaletteState>) {
    state = { ...state, ...next };
    listeners.forEach((listener) => listener());
}

export function openPalette(page: PalettePage = "root") {
    set({ isOpen: true, page });
}

export function closePalette() {
    set({ isOpen: false, page: "root" });
}

export function setPalettePage(page: PalettePage) {
    set({ page });
}

export function setPaletteContext(context: PaletteContext | null) {
    set({ context });
}

function subscribe(listener: () => void) {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

export function useCommandPalette(): PaletteState {
    return useSyncExternalStore(subscribe, () => state);
}
