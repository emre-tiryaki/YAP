import type { TranslationKey } from "../hooks/useTranslate";

export type ShortcutItem = {
    label: TranslationKey;
    // "mod" is rendered as Cmd on Mac and Ctrl elsewhere (see Kbd).
    keys: string[];
    // Registered with useShortcut, so the "shortcuts enabled" setting applies.
    global: boolean;
};

export type ShortcutGroup = {
    title: TranslationKey;
    items: ShortcutItem[];
};

// The global shortcuts are registered with useShortcut; saving a file, sending
// a comment and closing a dialog are handled by those components and always work.
export const shortcutGroups: ShortcutGroup[] = [
    {
        title: "settings.shortcuts.group.navigation",
        items: [
            { label: "settings.shortcuts.goHome", keys: ["G", "H"], global: true },
            { label: "settings.shortcuts.goTasks", keys: ["G", "T"], global: true },
            { label: "settings.shortcuts.goProjects", keys: ["G", "P"], global: true },
            { label: "settings.shortcuts.goResources", keys: ["G", "R"], global: true },
            { label: "settings.shortcuts.goSettings", keys: ["G", "S"], global: true },
        ],
    },
    {
        title: "settings.shortcuts.group.general",
        items: [
            { label: "settings.shortcuts.search", keys: ["mod", "K"], global: true },
            { label: "settings.shortcuts.help", keys: ["mod", "/"], global: true },
            { label: "settings.shortcuts.toggleSidebar", keys: ["mod", "B"], global: true },
            { label: "settings.shortcuts.openSettings", keys: ["mod", ","], global: true },
            { label: "settings.shortcuts.newTask", keys: ["N"], global: true },
            { label: "settings.shortcuts.closeDialog", keys: ["Esc"], global: false },
        ],
    },
    {
        title: "settings.shortcuts.group.tasks",
        items: [
            { label: "settings.shortcuts.changeStatus", keys: ["S"], global: true },
            { label: "settings.shortcuts.dueDate", keys: ["D"], global: true },
            { label: "settings.shortcuts.sendComment", keys: ["mod", "Enter"], global: false },
        ],
    },
    {
        title: "settings.shortcuts.group.resources",
        items: [
            { label: "settings.shortcuts.resourceRename", keys: ["F2"], global: true },
            { label: "settings.shortcuts.resourceTrash", keys: ["Delete"], global: true },
            { label: "settings.shortcuts.resourceCut", keys: ["mod", "X"], global: true },
            { label: "settings.shortcuts.resourceCopy", keys: ["mod", "C"], global: true },
            { label: "settings.shortcuts.resourcePaste", keys: ["mod", "V"], global: true },
            { label: "settings.shortcuts.saveFile", keys: ["mod", "S"], global: false },
        ],
    },
];
