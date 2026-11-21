import { updatePreferences, usePreferences } from "../hooks/usePreferences";
import { useTranslate } from "../hooks/useTranslate";
import { shortcutGroups } from "../util/shortcuts";
import { Kbd } from "./Kbd";
import { SettingRow } from "./SettingRow";
import { SettingsGroup } from "./SettingsGroup";
import { Toggle } from "./Toggle";

export function SettingsShortcuts() {
    const t = useTranslate();
    const { shortcutsEnabled } = usePreferences();

    return (
        <div className="flex flex-col gap-6">
            <SettingsGroup>
                <SettingRow
                    label={t("settings.shortcuts.enabled")}
                    hint={t("settings.shortcuts.enabled.hint")}
                >
                    <Toggle
                        label={t("settings.shortcuts.enabled")}
                        checked={shortcutsEnabled}
                        onChange={(enabled) =>
                            updatePreferences({ shortcutsEnabled: enabled })
                        }
                    />
                </SettingRow>
            </SettingsGroup>
            {shortcutGroups.map((group) => (
                <SettingsGroup key={group.title} title={t(group.title)}>
                    {group.items.map(({ label, keys, global }) => (
                        <SettingRow
                            key={label}
                            label={t(label)}
                            hint={
                                label === "settings.shortcuts.newTask"
                                    ? t("settings.shortcuts.newTask.hint")
                                    : undefined
                            }
                        >
                            <span
                                className={
                                    global && !shortcutsEnabled ? "opacity-40" : ""
                                }
                            >
                                <Kbd keys={keys} />
                            </span>
                        </SettingRow>
                    ))}
                </SettingsGroup>
            ))}
        </div>
    );
}
