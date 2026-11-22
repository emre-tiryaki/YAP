import { useId, useState, type FormEvent } from "react";
import {
    FiAlertCircle,
    FiArrowLeft,
    FiCheckCircle,
    FiEye,
    FiEyeOff,
    FiExternalLink,
} from "react-icons/fi";
import {
    aiErrorKey,
    useCreateAiConnection,
    useTestAiConnection,
    useUpdateAiConnection,
} from "../hooks/useAi";
import { useTranslate } from "../hooks/useTranslate";
import type { AiConnection, AiPreset } from "../types/ai";
import { aiPresetOrder, aiPresets } from "../util/aiPresets";
import { AiProviderIcon } from "./AiProviderIcon";
import { Button } from "./Button";
import { Dialog } from "./Dialog";
import { FormError } from "./FormError";
import { SubmitButton } from "./SubmitButton";

type AiConnectionDialogProps = {
    // null adds a connection.
    connection: AiConnection | null;
    onClose: () => void;
};

const fieldClass =
    "h-8 w-full rounded-md border border-line bg-layer-2 px-2.5 text-body-xs text-fg placeholder:text-fg-placeholder focus-visible:border-accent focus-visible:outline-none disabled:opacity-50";

export function AiConnectionDialog({
    connection,
    onClose,
}: AiConnectionDialogProps) {
    const t = useTranslate();
    // Adding starts on the provider grid; editing goes straight to the form.
    const [preset, setPreset] = useState<AiPreset | null>(
        connection?.preset ?? null,
    );

    return (
        <Dialog
            open
            onClose={onClose}
            title={t(
                connection ? "ai.connection.editTitle" : "ai.connection.addTitle",
            )}
        >
            {preset === null ? (
                <ProviderGrid onPick={setPreset} />
            ) : (
                <ConnectionForm
                    key={preset}
                    connection={connection}
                    preset={preset}
                    onBack={connection ? null : () => setPreset(null)}
                    onClose={onClose}
                />
            )}
        </Dialog>
    );
}

function ProviderGrid({ onPick }: { onPick: (preset: AiPreset) => void }) {
    const t = useTranslate();
    const groups: { titleKey: "ai.group.local" | "ai.group.api"; items: AiPreset[] }[] = [
        {
            titleKey: "ai.group.api",
            items: aiPresetOrder.filter((p) => !aiPresets[p].local && p !== "custom"),
        },
        {
            titleKey: "ai.group.local",
            items: aiPresetOrder.filter((p) => aiPresets[p].local || p === "custom"),
        },
    ];
    return (
        <div className="flex flex-col gap-4">
            <p className="text-body-xs text-fg-subtle">{t("ai.connection.pick")}</p>
            {groups.map((group) => (
                <section key={group.titleKey} aria-label={t(group.titleKey)}>
                    <h3 className="mb-2 text-caption font-medium text-fg-muted">
                        {t(group.titleKey)}
                    </h3>
                    <ul className="grid grid-cols-2 gap-2">
                        {group.items.map((p) => (
                            <li key={p}>
                                <button
                                    type="button"
                                    onClick={() => onPick(p)}
                                    className="flex w-full cursor-pointer items-center gap-2.5 rounded-lg border border-line bg-layer-2 p-2 text-left hover:border-line-strong hover:bg-layer-2-hover focus-visible:outline-2 focus-visible:outline-accent"
                                >
                                    <AiProviderIcon preset={p} size="sm" />
                                    <span className="truncate text-body-xs font-medium text-fg">
                                        {t(aiPresets[p].labelKey)}
                                    </span>
                                </button>
                            </li>
                        ))}
                    </ul>
                </section>
            ))}
        </div>
    );
}

type ConnectionFormProps = {
    connection: AiConnection | null;
    preset: AiPreset;
    onBack: (() => void) | null;
    onClose: () => void;
};

function ConnectionForm({
    connection,
    preset,
    onBack,
    onClose,
}: ConnectionFormProps) {
    const t = useTranslate();
    const ids = useId();
    const create = useCreateAiConnection();
    const update = useUpdateAiConnection();
    const test = useTestAiConnection(connection);
    const info = aiPresets[preset];

    const [name, setName] = useState(connection?.name ?? t(info.labelKey));
    const [baseUrl, setBaseUrl] = useState(connection?.baseUrl ?? info.baseUrl);
    const [apiKey, setApiKey] = useState("");
    const [reveal, setReveal] = useState(false);
    const [removeKey, setRemoveKey] = useState(false);
    // A saved key stays hidden behind "Replace" until asked for.
    const [replacing, setReplacing] = useState(false);
    const [advanced, setAdvanced] = useState(
        connection !== null || preset === "custom" || info.local,
    );

    const isPending = create.isPending || update.isPending;
    const error = create.error ?? update.error;
    const hasSavedKey = connection?.hasKey === true && !removeKey;
    const showKeyInput = !hasSavedKey || replacing;
    const keyMissing = info.keyRequired && apiKey === "" && !hasSavedKey;
    const canSubmit = name.trim() !== "" && baseUrl.trim() !== "" && !keyMissing;

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (connection) {
            update.mutate(
                {
                    id: connection.id,
                    input: {
                        name,
                        baseUrl,
                        apiKey: removeKey ? "" : apiKey !== "" ? apiKey : undefined,
                    },
                },
                { onSuccess: onClose },
            );
        } else {
            create.mutate({ name, preset, baseUrl, apiKey }, { onSuccess: onClose });
        }
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex items-center gap-3">
                <AiProviderIcon preset={preset} />
                <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-semibold text-fg">
                        {t(info.labelKey)}
                    </p>
                    <p className="text-caption text-fg-muted">
                        {t(info.local ? "ai.connection.localHint" : "ai.connection.apiHint")}
                    </p>
                </div>
                {onBack && (
                    <Button variant="ghost" size="md" leadingIcon={FiArrowLeft} onClick={onBack}>
                        {t("ai.connection.change")}
                    </Button>
                )}
            </div>

            <div className="flex flex-col gap-1.5">
                <div className="flex items-baseline justify-between gap-2">
                    <label htmlFor={`${ids}-key`} className="text-body-xs font-medium text-fg">
                        {t("ai.connection.apiKey")}
                        {!info.keyRequired && (
                            <span className="ml-1.5 font-normal text-fg-subtle">
                                {t("ai.connection.optional")}
                            </span>
                        )}
                    </label>
                    {info.keyUrl && (
                        <a
                            href={info.keyUrl}
                            target="_blank"
                            rel="noreferrer noopener"
                            className="inline-flex items-center gap-1 text-caption text-fg-muted underline hover:text-fg"
                        >
                            {t("ai.connection.getKey")}
                            <FiExternalLink aria-hidden />
                        </a>
                    )}
                </div>
                {showKeyInput && !removeKey ? (
                    <div className="relative">
                        <input
                            id={`${ids}-key`}
                            type={reveal ? "text" : "password"}
                            autoComplete="off"
                            spellCheck={false}
                            value={apiKey}
                            placeholder={info.keyRequired ? "sk-…" : ""}
                            onChange={(event) => {
                                setApiKey(event.target.value);
                                test.reset();
                            }}
                            className={`${fieldClass} pr-9 font-mono`}
                        />
                        <button
                            type="button"
                            aria-label={t(reveal ? "ai.connection.hideKey" : "ai.connection.showKey")}
                            aria-pressed={reveal}
                            onClick={() => setReveal(!reveal)}
                            className="absolute top-1/2 right-1 flex size-6 -translate-y-1/2 cursor-pointer items-center justify-center rounded text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                        >
                            {reveal ? <FiEyeOff aria-hidden /> : <FiEye aria-hidden />}
                        </button>
                    </div>
                ) : (
                    <div
                        id={`${ids}-key`}
                        className="flex h-8 items-center justify-between gap-2 rounded-md border border-line bg-layer-1 pr-1 pl-2.5"
                    >
                        <span className="flex items-center gap-1.5 text-body-xs text-fg-muted">
                            {removeKey ? (
                                t("ai.connection.apiKey.removed")
                            ) : (
                                <>
                                    <FiCheckCircle aria-hidden className="text-success" />
                                    <span className="font-mono tracking-widest">••••••••</span>
                                    {t("ai.connection.apiKey.stored")}
                                </>
                            )}
                        </span>
                        <span className="flex gap-1">
                            {removeKey ? (
                                <Button variant="ghost" size="sm" onClick={() => setRemoveKey(false)}>
                                    {t("ai.connection.undo")}
                                </Button>
                            ) : (
                                <>
                                    <Button variant="ghost" size="sm" onClick={() => setReplacing(true)}>
                                        {t("ai.connection.replace")}
                                    </Button>
                                    <Button variant="ghost" size="sm" onClick={() => setRemoveKey(true)}>
                                        {t("ai.connection.apiKey.remove")}
                                    </Button>
                                </>
                            )}
                        </span>
                    </div>
                )}
                <p className="text-caption text-fg-subtle">{t("ai.connection.keychain")}</p>
            </div>

            {advanced ? (
                <div className="grid gap-3 sm:grid-cols-2">
                    <div className="flex flex-col gap-1.5">
                        <label htmlFor={`${ids}-name`} className="text-body-xs font-medium text-fg">
                            {t("ai.connection.name")}
                        </label>
                        <input
                            id={`${ids}-name`}
                            value={name}
                            maxLength={100}
                            required
                            onChange={(event) => setName(event.target.value)}
                            className={fieldClass}
                        />
                    </div>
                    <div className="flex flex-col gap-1.5">
                        <label htmlFor={`${ids}-url`} className="text-body-xs font-medium text-fg">
                            {t("ai.connection.baseUrl")}
                        </label>
                        <input
                            id={`${ids}-url`}
                            value={baseUrl}
                            required
                            inputMode="url"
                            spellCheck={false}
                            onChange={(event) => {
                                setBaseUrl(event.target.value);
                                test.reset();
                            }}
                            className={`${fieldClass} font-mono`}
                        />
                    </div>
                </div>
            ) : (
                <button
                    type="button"
                    onClick={() => setAdvanced(true)}
                    className="cursor-pointer self-start text-caption text-fg-muted underline hover:text-fg"
                >
                    {t("ai.connection.advanced")}
                </button>
            )}

            <div className="flex min-h-8 items-center gap-3 rounded-lg border border-line bg-layer-1 px-2 py-1.5">
                <Button
                    size="md"
                    loading={test.isPending}
                    disabled={baseUrl.trim() === "" || keyMissing}
                    onClick={() => test.mutate({ preset, baseUrl, apiKey })}
                >
                    {t(test.isPending ? "ai.connection.testing" : "ai.connection.test")}
                </Button>
                <div role="status" className="min-w-0 text-body-xs">
                    {test.isSuccess && (
                        <p className="flex items-center gap-1.5 text-success">
                            <FiCheckCircle aria-hidden className="shrink-0" />
                            {t("ai.connection.test.ok").replace("{n}", String(test.data.length))}
                        </p>
                    )}
                    {test.isError && (
                        <p className="flex items-center gap-1.5 text-danger">
                            <FiAlertCircle aria-hidden className="shrink-0" />
                            {t(aiErrorKey(test.error))}
                        </p>
                    )}
                    {test.isIdle && (
                        <p className="text-fg-subtle">{t("ai.connection.test.hint")}</p>
                    )}
                </div>
            </div>

            {error && <FormError message={t(aiErrorKey(error))} />}
            <div className="flex justify-end gap-2">
                <Button size="lg" onClick={onClose}>
                    {t("common.cancel")}
                </Button>
                <SubmitButton disabled={isPending || !canSubmit}>
                    {t("common.save")}
                </SubmitButton>
            </div>
        </form>
    );
}
