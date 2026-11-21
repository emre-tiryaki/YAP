import { type KeyboardEvent as ReactKeyboardEvent, useState } from "react";
import { FiUser } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import { useUpdateWorkspace, useWorkspace } from "../hooks/useWorkspace";
import { type UpdateWorkspaceInput } from "../types/workspace";
import { SettingsGroup } from "./SettingsGroup";
import { SettingRow } from "./SettingRow";
import { SettingsData } from "./SettingsData";

export function SettingsAccount() {
    const t = useTranslate();
    const workspace = useWorkspace();
    const username = workspace.data?.username ?? "";

    return (
        <div className="flex flex-col gap-6">
            <SettingsGroup>
                <div className="flex items-center gap-4 px-5 py-4">
                    <span
                        aria-hidden
                        className="flex size-14 shrink-0 items-center justify-center rounded-full bg-gray-900 text-xl font-semibold text-white"
                    >
                        {username ? username[0].toUpperCase() : <FiUser />}
                    </span>
                    <div className="min-w-0">
                        <p className="truncate font-medium text-gray-900">{username}</p>
                        <p className="truncate text-sm text-gray-500">
                            {workspace.data?.name}
                        </p>
                    </div>
                </div>
                {workspace.data && (
                    <>
                        <NameSetting
                            label={t("settings.account.username")}
                            value={workspace.data.username}
                            toInput={(username) => ({ username })}
                        />
                        <NameSetting
                            label={t("settings.account.workspace")}
                            value={workspace.data.name}
                            toInput={(name) => ({ name })}
                        />
                    </>
                )}
            </SettingsGroup>

            <SettingsData />
        </div>
    );
}

// A name saved when the field loses focus or Enter is pressed.
function NameSetting({
    label,
    value,
    toInput,
}: {
    label: string;
    value: string;
    toInput: (name: string) => UpdateWorkspaceInput;
}) {
    const t = useTranslate();
    const update = useUpdateWorkspace();
    const [draft, setDraft] = useState(value);
    const [error, setError] = useState<string | null>(null);
    const [isSaved, setIsSaved] = useState(false);

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
        update.mutate(toInput(name), {
            onSuccess: () => {
                setDraft(name);
                setIsSaved(true);
            },
            onError: () => {
                setDraft(value);
                setError(t("settings.account.saveError"));
            },
        });
    }

    function handleKeyDown(event: ReactKeyboardEvent<HTMLInputElement>) {
        if (event.key === "Enter") event.currentTarget.blur();
    }

    return (
        <SettingRow
            label={label}
            hint={error ?? (isSaved ? t("settings.account.saved") : undefined)}
            hintTone={error ? "error" : "default"}
        >
            <input
                value={draft}
                onChange={(event) => {
                    setDraft(event.target.value);
                    setIsSaved(false);
                }}
                onBlur={save}
                onKeyDown={handleKeyDown}
                disabled={update.isPending}
                maxLength={100}
                aria-label={label}
                aria-invalid={error !== null}
                className="w-56 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm aria-invalid:border-red-400 disabled:opacity-60"
            />
        </SettingRow>
    );
}
