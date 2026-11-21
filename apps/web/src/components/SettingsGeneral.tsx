import { ViewToggle } from "./ViewToggle";
import { useDateFormat } from "../hooks/useDateFormat";
import { updatePreferences, usePreferences } from "../hooks/usePreferences";
import { useTranslate } from "../hooks/useTranslate";
import { defaultUserSettings, useUpdateUserSettings, useUserSettings } from "../hooks/useUserSettings";
import { type DateFormat, type Language, type TimeFormat, type WeekStart } from "../types/settings";
import { formatDate } from "../util/date";
import { PropertySelect } from "./PropertySelect";
import { SettingsGroup } from "./SettingsGroup";
import { SettingRow } from "./SettingRow";
import { Toggle } from "./Toggle";

export function SettingsGeneral() {
    const t = useTranslate();
    const preferences = usePreferences();
    const format = useDateFormat();
    const now = new Date();
    const dateFormats: DateFormat[] = ["dmy", "mdy", "iso"];

    return (
        <div className="flex flex-col gap-6">
            <SettingsGroup>
                <SettingRow
                    label={t("settings.general.language")}
                    hint={t("settings.general.language.hint")}
                >
                    <PropertySelect
                        variant="ghost"
                        placeholder=""
                        ariaLabel={t("settings.general.language")}
                        value={preferences.language}
                        onChange={(language) =>
                            updatePreferences({ language: language as Language })
                        }
                        options={[
                            { value: "en", label: "English" },
                            { value: "tr", label: "Türkçe" },
                        ]}
                    />
                </SettingRow>
            </SettingsGroup>

            <SettingsGroup title={t("settings.general.dateTime")}>
                <SettingRow
                    label={t("settings.general.weekStart")}
                    hint={t("settings.general.weekStart.hint")}
                >
                    <ViewToggle
                        label={t("settings.general.weekStart")}
                        value={preferences.weekStart}
                        onChange={(weekStart) =>
                            updatePreferences({ weekStart: weekStart as WeekStart })
                        }
                        options={[
                            { view: "monday", label: t("settings.general.weekStart.monday") },
                            { view: "sunday", label: t("settings.general.weekStart.sunday") },
                        ]}
                    />
                </SettingRow>
                <SettingRow label={t("settings.general.dateFormat")}>
                    <PropertySelect
                        variant="ghost"
                        placeholder=""
                        ariaLabel={t("settings.general.dateFormat")}
                        value={preferences.dateFormat}
                        onChange={(dateFormat) =>
                            updatePreferences({ dateFormat: dateFormat as DateFormat })
                        }
                        options={dateFormats.map((value) => ({
                            value,
                            label: formatDate(now, value),
                        }))}
                    />
                </SettingRow>
                <SettingRow
                    label={t("settings.general.timeFormat")}
                    hint={`${t("settings.general.example")}: ${format.dateTime(now)}`}
                >
                    <ViewToggle
                        label={t("settings.general.timeFormat")}
                        value={preferences.timeFormat}
                        onChange={(timeFormat) =>
                            updatePreferences({ timeFormat: timeFormat as TimeFormat })
                        }
                        options={[
                            { view: "24h", label: t("settings.general.timeFormat.24h") },
                            { view: "12h", label: t("settings.general.timeFormat.12h") },
                        ]}
                    />
                </SettingRow>
            </SettingsGroup>

            <EditorSettings />
        </div>
    );
}

const autosaveDelays = [500, 1000, 2000, 5000];

// Stored in the database, so they follow the user to every device.
function EditorSettings() {
    const t = useTranslate();
    const settings = useUserSettings();
    const update = useUpdateUserSettings();
    const current = settings.data ?? defaultUserSettings;
    const delays = autosaveDelays.includes(current.autosaveDelayMs)
        ? autosaveDelays
        : [...autosaveDelays, current.autosaveDelayMs].sort((a, b) => a - b);

    return (
        <SettingsGroup title={t("settings.editor.title")}>
            <SettingRow
                label={t("settings.editor.autosave")}
                hint={
                    update.isError
                        ? t("common.error")
                        : t("settings.editor.autosave.hint")
                }
                hintTone={update.isError ? "error" : "default"}
            >
                <Toggle
                    label={t("settings.editor.autosave")}
                    checked={current.autosaveEnabled}
                    onChange={(autosaveEnabled) => update.mutate({ autosaveEnabled })}
                />
            </SettingRow>
            <SettingRow
                label={t("settings.editor.delay")}
                hint={t("settings.editor.delay.hint")}
            >
                <div className={current.autosaveEnabled ? "" : "opacity-40"}>
                    <PropertySelect
                        variant="ghost"
                        placeholder=""
                        ariaLabel={t("settings.editor.delay")}
                        value={String(current.autosaveDelayMs)}
                        onChange={(value) =>
                            update.mutate({ autosaveDelayMs: Number(value) })
                        }
                        options={delays.map((ms) => ({
                            value: String(ms),
                            label: `${ms / 1000} ${t("settings.editor.seconds")}`,
                        }))}
                    />
                </div>
            </SettingRow>
        </SettingsGroup>
    );
}
