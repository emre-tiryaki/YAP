import { useState } from "react";
import { useTranslate } from "../hooks/useTranslate";
import { defaultUserSettings, useUpdateUserSettings, useUserSettings } from "../hooks/useUserSettings";
import { type StatusCategory } from "../types/task";
import { SettingsGroup } from "./SettingsGroup";
import { SettingRow } from "./SettingRow";
import { Toggle } from "./Toggle";

const categoryDefaults: { category: StatusCategory; color: string }[] = [
    { category: "todo", color: "#9ca3af" },
    { category: "in_progress", color: "#3b82f6" },
    { category: "done", color: "#22c55e" },
];

export function SettingsCategories() {
    const t = useTranslate();
    const settings = useUserSettings();
    const update = useUpdateUserSettings();
    const current = settings.data ?? defaultUserSettings;

    return (
        <div className="flex flex-col gap-6">
            <SettingsGroup title={t("settings.categories.defaults")}>
                {settings.data &&
                    categoryDefaults.map(({ category, color }) => (
                        <StatusNameSetting
                            key={category}
                            category={category}
                            color={color}
                            value={settings.data.statusNames[category]}
                        />
                    ))}
            </SettingsGroup>

            <SettingsGroup>
                <SettingRow
                    label={t("settings.categories.doneToBottom")}
                    hint={t("settings.categories.doneToBottom.hint")}
                >
                    <div className={current.hideDone ? "opacity-40" : ""}>
                        <Toggle
                            label={t("settings.categories.doneToBottom")}
                            checked={current.doneToBottom}
                            onChange={(doneToBottom) => update.mutate({ doneToBottom })}
                        />
                    </div>
                </SettingRow>
                <SettingRow
                    label={t("settings.categories.hideDone")}
                    hint={
                        update.isError
                            ? t("common.error")
                            : t("settings.categories.hideDone.hint")
                    }
                    hintTone={update.isError ? "error" : "default"}
                >
                    <Toggle
                        label={t("settings.categories.hideDone")}
                        checked={current.hideDone}
                        onChange={(hideDone) => update.mutate({ hideDone })}
                    />
                </SettingRow>
            </SettingsGroup>
        </div>
    );
}

// A default status name, saved when the field loses focus or Enter is
// pressed.
function StatusNameSetting({
    category,
    color,
    value,
}: {
    category: StatusCategory;
    color: string;
    value: string;
}) {
    const t = useTranslate();
    const update = useUpdateUserSettings();
    const [draft, setDraft] = useState(value);
    const [error, setError] = useState<string | null>(null);
    const [isSaved, setIsSaved] = useState(false);
    const label = `${t("settings.categories.defaultName")}: ${t(`status.category.${category}`)}`;

    function save() {
        const name = draft.trim();
        setIsSaved(false);
        if (name === "") {
            setDraft(value);
            setError(t("settings.account.nameRequired"));
            return;
        }
        setError(null);
        if (name === value) {
            setDraft(value);
            return;
        }
        update.mutate(
            { statusNames: { [category]: name } },
            {
                onSuccess: (settings) => {
                    setDraft(settings.statusNames[category]);
                    setIsSaved(true);
                },
                onError: () => {
                    setDraft(value);
                    setError(t("settings.account.saveError"));
                },
            },
        );
    }

    let hint = t(`settings.categories.${category}.hint`);
    if (isSaved) hint = t("settings.account.saved");
    if (error) hint = error;

    return (
        <div className="flex items-center gap-4 px-5 py-4">
            <span
                aria-hidden
                style={{ backgroundColor: color }}
                className="size-3 shrink-0 rounded-full"
            />
            <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-gray-900">
                    {t(`status.category.${category}`)}
                </p>
                <p
                    role={error ? "alert" : undefined}
                    className={`text-xs ${error ? "text-red-600" : "text-gray-500"}`}
                >
                    {hint}
                </p>
            </div>
            <input
                value={draft}
                onChange={(event) => {
                    setDraft(event.target.value);
                    setIsSaved(false);
                }}
                onBlur={save}
                onKeyDown={(event) => {
                    if (event.key === "Enter") event.currentTarget.blur();
                }}
                disabled={update.isPending}
                maxLength={100}
                aria-label={label}
                aria-invalid={error !== null}
                placeholder={t("settings.categories.defaultName")}
                className="w-44 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm aria-invalid:border-red-400 disabled:opacity-60"
            />
        </div>
    );
}
