import { useState } from "react";
import { FiCheckCircle, FiMoreHorizontal, FiPlus } from "react-icons/fi";
import {
    aiErrorKey,
    useAiConnections,
    useDeleteAiConnection,
} from "../hooks/useAi";
import { useTranslate } from "../hooks/useTranslate";
import {
    useCurrentUserSettings,
    useUpdateUserSettings,
} from "../hooks/useUserSettings";
import type { AiConnection } from "../types/ai";
import { aiPresets } from "../util/aiPresets";
import { AiConnectionDialog } from "./AiConnectionDialog";
import { AiProviderIcon } from "./AiProviderIcon";
import { Button } from "./Button";
import { ConfirmDialog } from "./ConfirmDialog";
import { FormError } from "./FormError";
import { InlineError } from "./InlineError";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";
import { Pill } from "./Pill";
import { SettingsGroup } from "./SettingsGroup";

const MAX_INSTRUCTIONS = 4000;

export function SettingsAI() {
    const t = useTranslate();
    const connections = useAiConnections();
    const remove = useDeleteAiConnection();
    // undefined: closed; null: adding; otherwise editing that connection.
    const [editing, setEditing] = useState<AiConnection | null | undefined>(
        undefined,
    );
    const [deleting, setDeleting] = useState<AiConnection | null>(null);

    return (
        <div className="flex flex-col gap-8">
            <SettingsGroup title={t("settings.ai.connections")}>
                {connections.isError ? (
                    <div className="p-4">
                        <InlineError
                            message={t("common.error")}
                            onRetry={() => void connections.refetch()}
                        />
                    </div>
                ) : (connections.data ?? []).length === 0 ? (
                    <p className="px-5 py-4 text-body-xs text-fg-subtle">
                        {connections.isPending
                            ? t("common.loading")
                            : t("settings.ai.connections.empty")}
                    </p>
                ) : (
                    (connections.data ?? []).map((c) => (
                        <div
                            key={c.id}
                            className="flex items-center gap-3 px-4 py-3"
                        >
                            <AiProviderIcon preset={c.preset} />
                            <div className="min-w-0 flex-1">
                                <p className="truncate text-sm font-medium text-fg">
                                    {c.name}
                                </p>
                                <p className="truncate text-caption text-fg-muted">
                                    {t(aiPresets[c.preset].labelKey)} ·{" "}
                                    <span className="font-mono">{c.baseUrl}</span>
                                </p>
                            </div>
                            <div className="flex shrink-0 items-center gap-2">
                                {c.hasKey ? (
                                    <Pill
                                        icon={<FiCheckCircle aria-hidden />}
                                        color="text-success"
                                    >
                                        {t("settings.ai.keySaved")}
                                    </Pill>
                                ) : aiPresets[c.preset].local ? (
                                    <Pill>{t("settings.ai.local")}</Pill>
                                ) : (
                                    <Pill className="border-warning text-warning">
                                        {t("settings.ai.noKey")}
                                    </Pill>
                                )}
                                <Menu
                                    placement="bottom-end"
                                    trigger={(props) => (
                                        <Button
                                            variant="ghost"
                                            size="md"
                                            aria-label={`${t("settings.ai.menu")}: ${c.name}`}
                                            {...props}
                                        >
                                            <FiMoreHorizontal aria-hidden />
                                        </Button>
                                    )}
                                >
                                    <MenuItem
                                        label={t("settings.ai.edit")}
                                        onSelect={() => setEditing(c)}
                                    />
                                    <MenuItem
                                        danger
                                        label={t("settings.ai.delete")}
                                        onSelect={() => setDeleting(c)}
                                    />
                                </Menu>
                            </div>
                        </div>
                    ))
                )}
                <div className="px-5 py-3">
                    <Button
                        leadingIcon={FiPlus}
                        onClick={() => setEditing(null)}
                    >
                        {t("settings.ai.add")}
                    </Button>
                </div>
            </SettingsGroup>

            <InstructionsGroup />

            {editing !== undefined && (
                <AiConnectionDialog
                    // Keyed so each opening starts from its own connection.
                    key={editing?.id ?? "new"}
                    connection={editing}
                    onClose={() => setEditing(undefined)}
                />
            )}
            <ConfirmDialog
                isOpen={deleting !== null}
                title={t("settings.ai.delete.title")}
                message={t("settings.ai.delete.message")}
                confirmLabel={t("settings.ai.delete")}
                isPending={remove.isPending}
                isError={remove.isError}
                errorMessage={t(aiErrorKey(remove.error))}
                onConfirm={() =>
                    deleting &&
                    remove.mutate(deleting.id, {
                        onSuccess: () => setDeleting(null),
                    })
                }
                onClose={() => {
                    setDeleting(null);
                    remove.reset();
                }}
            />
        </div>
    );
}

// The personal instructions are saved when the field loses focus.
function InstructionsGroup() {
    const t = useTranslate();
    const settings = useCurrentUserSettings();
    const update = useUpdateUserSettings();
    // null: not edited, show the stored text.
    const [draft, setDraft] = useState<string | null>(null);
    const value = draft ?? settings.aiInstructions;

    function save() {
        if (draft === null || draft === settings.aiInstructions) {
            setDraft(null);
            return;
        }
        update.mutate(
            { aiInstructions: draft },
            { onSuccess: () => setDraft(null) },
        );
    }

    return (
        <SettingsGroup title={t("settings.ai.instructions")}>
            <div className="flex flex-col gap-2 px-5 py-4">
                <label className="text-sm font-medium text-fg" htmlFor="ai-instructions">
                    {t("settings.ai.instructions.label")}
                </label>
                <textarea
                    id="ai-instructions"
                    value={value}
                    rows={5}
                    maxLength={MAX_INSTRUCTIONS}
                    placeholder={t("settings.ai.instructions.placeholder")}
                    onChange={(event) => setDraft(event.target.value)}
                    onBlur={save}
                    className="w-full resize-y rounded-md border border-line bg-layer-2 px-3 py-2 text-body-xs text-fg focus-visible:outline-2 focus-visible:outline-accent"
                />
                <div className="flex items-center justify-between gap-4 text-caption text-fg-muted">
                    <span>{t("settings.ai.instructions.hint")}</span>
                    <span>
                        {t("settings.ai.instructions.count").replace(
                            "{n}",
                            String([...value].length),
                        )}
                    </span>
                </div>
                {update.isError && (
                    <FormError message={t(aiErrorKey(update.error))} />
                )}
            </div>
        </SettingsGroup>
    );
}
