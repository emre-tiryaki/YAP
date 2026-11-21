import { useState } from "react";
import { useTranslate } from "../hooks/useTranslate";
import { defaultUserSettings, useUpdateUserSettings, useUserSettings } from "../hooks/useUserSettings";
import { type NotificationSettings as NotificationPreferences, type Reminder } from "../types/userSettings";
import { playNotificationSound } from "../util/notificationSound";
import { PropertySelect } from "./PropertySelect";
import { SettingsGroup } from "./SettingsGroup";
import { SettingRow } from "./SettingRow";
import { Toggle } from "./Toggle";

export function SettingsNotifications() {
    const t = useTranslate();
    const settings = useUserSettings();
    const update = useUpdateUserSettings();
    const current = (settings.data ?? defaultUserSettings).notifications;
    const save = (patch: Partial<NotificationPreferences>) =>
        update.mutate({ notifications: patch });

    return (
        <div className="flex flex-col gap-6">
            <SettingsGroup>
                <DesktopNotificationSetting />
            </SettingsGroup>

            <SettingsGroup title={t("settings.notifications.tasks")}>
                <SettingRow
                    label={t("settings.notifications.dueSoon")}
                    hint={
                        update.isError
                            ? t("common.error")
                            : t("settings.notifications.dueSoon.hint")
                    }
                    hintTone={update.isError ? "error" : "default"}
                >
                    <Toggle
                        label={t("settings.notifications.dueSoon")}
                        checked={current.dueSoon}
                        onChange={(dueSoon) => save({ dueSoon })}
                    />
                </SettingRow>
                <SettingRow label={t("settings.notifications.reminder")}>
                    <div className={current.dueSoon ? "" : "opacity-40"}>
                        <PropertySelect
                            variant="ghost"
                            placeholder=""
                            ariaLabel={t("settings.notifications.reminder")}
                            value={current.reminder}
                            onChange={(reminder) => save({ reminder: reminder as Reminder })}
                            options={reminders.map((value) => ({
                                value,
                                label: t(`settings.notifications.reminder.${value}`),
                            }))}
                        />
                    </div>
                </SettingRow>
                <SettingRow
                    label={t("settings.notifications.overdue")}
                    hint={t("settings.notifications.overdue.hint")}
                >
                    <Toggle
                        label={t("settings.notifications.overdue")}
                        checked={current.overdue}
                        onChange={(overdue) => save({ overdue })}
                    />
                </SettingRow>
            </SettingsGroup>

            <SettingsGroup title={t("settings.notifications.activity")}>
                <SettingRow
                    label={t("settings.notifications.comments")}
                    hint={t("settings.notifications.comments.hint")}
                >
                    <Toggle
                        label={t("settings.notifications.comments")}
                        checked={current.comments}
                        onChange={(comments) => save({ comments })}
                    />
                </SettingRow>
                <SettingRow
                    label={t("settings.notifications.sound")}
                    hint={t("settings.notifications.sound.hint")}
                >
                    <Toggle
                        label={t("settings.notifications.sound")}
                        checked={current.sound}
                        onChange={(sound) => {
                            save({ sound });
                            if (sound) playNotificationSound();
                        }}
                    />
                </SettingRow>
            </SettingsGroup>
        </div>
    );
}

const reminders: Reminder[] = ["1h", "1d", "1w"];

type Permission = NotificationPermission | "unsupported";

function readPermission(): Permission {
    return "Notification" in window ? Notification.permission : "unsupported";
}

// System notifications are allowed per browser, so this is not a stored
// setting: it shows and asks for the browser's permission.
function DesktopNotificationSetting() {
    const t = useTranslate();
    const [permission, setPermission] = useState<Permission>(readPermission);

    return (
        <SettingRow
            label={t("settings.notifications.desktop")}
            hint={t(`settings.notifications.desktop.${permission}`)}
        >
            {permission === "default" ? (
                <button
                    type="button"
                    onClick={() =>
                        void Notification.requestPermission().then(setPermission)
                    }
                    className="cursor-pointer rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                    {t("settings.notifications.desktop.allow")}
                </button>
            ) : (
                <span
                    className={`text-sm font-medium ${
                        permission === "granted" ? "text-green-600" : "text-gray-400"
                    }`}
                >
                    {t(
                        permission === "granted"
                            ? "settings.notifications.desktop.on"
                            : "settings.notifications.desktop.off",
                    )}
                </span>
            )}
        </SettingRow>
    );
}
