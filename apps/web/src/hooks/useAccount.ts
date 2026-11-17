import { useMutation, useQueryClient } from "@tanstack/react-query";
import { importAccount, resetAccount } from "../api/client";

// Device-only state (pins, preferences, layout) lives under this prefix.
const LOCAL_STORAGE_PREFIX = "yap.";

function clearLocalState() {
    const keys: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key?.startsWith(LOCAL_STORAGE_PREFIX)) keys.push(key);
    }
    keys.forEach((key) => localStorage.removeItem(key));
}

export function useResetAccount() {
    return useMutation({
        mutationFn: resetAccount,
        onSuccess: () => {
            clearLocalState();
            // A full reload drops every cached query and in-memory store, so
            // the app starts like on a first run.
            window.location.assign("/home");
        },
    });
}

export function useImportAccount() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: importAccount,
        // An import can touch every list, the names and the settings.
        onSuccess: () => queryClient.invalidateQueries(),
    });
}
