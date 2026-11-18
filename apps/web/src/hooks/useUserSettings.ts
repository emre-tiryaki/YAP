import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getUserSettings, updateUserSettings } from "../api/client";
import type { UserSettings } from "../types/userSettings";

const settingsKey = ["settings"] as const;

// Used until the stored settings have loaded. Must match the backend's
// defaults (service.defaultUserSettings).
export const defaultUserSettings: UserSettings = {
    autosaveEnabled: true,
    autosaveDelayMs: 1000,
    hideDone: false,
    doneToBottom: false,
    statusNames: { todo: "To Do", in_progress: "In Progress", done: "Done" },
    notifications: {
        dueSoon: true,
        reminder: "1d",
        overdue: true,
        comments: false,
        sound: false,
    },
    repositoriesAutoUpdate: true,
};

export function useUserSettings() {
    return useQuery({ queryKey: settingsKey, queryFn: getUserSettings });
}

// The stored settings, or the defaults while they load.
export function useCurrentUserSettings(): UserSettings {
    return useUserSettings().data ?? defaultUserSettings;
}

export function useUpdateUserSettings() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: updateUserSettings,
        onSuccess: (settings) =>
            queryClient.setQueryData(settingsKey, settings),
    });
}
