import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getUserSettings, updateUserSettings } from "../api/client";
import type { UserSettings } from "../types/userSettings";

const settingsKey = ["settings"] as const;

// Used until the stored settings have loaded.
export const defaultUserSettings: UserSettings = {
    autosaveEnabled: true,
    autosaveDelayMs: 1000,
};

export function useUserSettings() {
    return useQuery({ queryKey: settingsKey, queryFn: getUserSettings });
}

export function useUpdateUserSettings() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: updateUserSettings,
        onSuccess: (settings) =>
            queryClient.setQueryData(settingsKey, settings),
    });
}
