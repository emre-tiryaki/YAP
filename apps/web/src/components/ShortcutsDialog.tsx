import { closeShortcutsDialog, useShortcutsDialog } from "../hooks/useShortcutsDialog";
import { useTranslate } from "../hooks/useTranslate";
import { shortcutGroups } from "../util/shortcuts";
import { Dialog } from "./Dialog";
import { Kbd } from "./Kbd";

// Lists every keyboard shortcut; opened with mod+/ or from the palette.
export function ShortcutsDialog() {
    const t = useTranslate();
    const isOpen = useShortcutsDialog();

    return (
        <Dialog
            open={isOpen}
            onClose={closeShortcutsDialog}
            title={t("settings.shortcuts.help")}
            size="lg"
        >
            <div className="grid gap-6 sm:grid-cols-2">
                {shortcutGroups.map((group) => (
                    <section key={group.title} aria-label={t(group.title)}>
                        <h3 className="mb-2 text-caption font-semibold text-fg-subtle">
                            {t(group.title)}
                        </h3>
                        <ul className="flex flex-col">
                            {group.items.map((item) => (
                                <li
                                    key={item.label}
                                    className="flex min-h-8 items-center justify-between gap-4 text-body-xs text-fg"
                                >
                                    {t(item.label)}
                                    <Kbd keys={item.keys} />
                                </li>
                            ))}
                        </ul>
                    </section>
                ))}
            </div>
        </Dialog>
    );
}
