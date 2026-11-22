import type { DisplayOptions } from "../types/taskView";
import {
    defaultDisplay,
    updatePreferences,
    getPreferences,
    usePreferences,
} from "./usePreferences";

// The display options (grouping, order, properties) of one plan, kept per
// device in the preferences.
export function usePlanDisplay(planId: string): DisplayOptions {
    const { planDisplay } = usePreferences();
    return planDisplay[planId] ?? defaultDisplay();
}

export function updatePlanDisplay(
    planId: string,
    patch: Partial<DisplayOptions>,
) {
    const { planDisplay } = getPreferences();
    updatePreferences({
        planDisplay: {
            ...planDisplay,
            [planId]: { ...(planDisplay[planId] ?? defaultDisplay()), ...patch },
        },
    });
}
