import { useState, type FormEvent } from "react";
import {
    aiErrorKey,
    useCreateAiConnection,
    useTestAiConnection,
    useUpdateAiConnection,
} from "../hooks/useAi";
import { useTranslate } from "../hooks/useTranslate";
import type { AiConnection, AiPreset } from "../types/ai";
import { aiPresetOrder, aiPresets } from "../util/aiPresets";
import { Button } from "./Button";
import { Dialog } from "./Dialog";
import { FormError } from "./FormError";
import { SelectInput } from "./SelectInput";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

type AiConnectionDialogProps = {
    // null adds a connection.
    connection: AiConnection | null;
    onClose: () => void;
};

export function AiConnectionDialog({
    connection,
    onClose,
}: AiConnectionDialogProps) {
    const t = useTranslate();
    const create = useCreateAiConnection();
    const update = useUpdateAiConnection();
    const test = useTestAiConnection(connection);

    const [preset, setPreset] = useState<AiPreset>(
        connection?.preset ?? "ollama",
    );
    const [name, setName] = useState(
        connection?.name ?? t(aiPresets.ollama.labelKey),
    );
    const [baseUrl, setBaseUrl] = useState(
        connection?.baseUrl ?? aiPresets.ollama.baseUrl,
    );
    const [apiKey, setApiKey] = useState("");
    const [removeKey, setRemoveKey] = useState(false);
    // Changing the provider fills the name and URL, unless the user typed them.
    const [nameEdited, setNameEdited] = useState(false);
    const [urlEdited, setUrlEdited] = useState(false);

    const info = aiPresets[preset];
    const isPending = create.isPending || update.isPending;
    const error = create.error ?? update.error;
    const hasSavedKey = connection?.hasKey === true && !removeKey;

    function choosePreset(next: AiPreset) {
        setPreset(next);
        if (!nameEdited) setName(t(aiPresets[next].labelKey));
        if (!urlEdited) setBaseUrl(aiPresets[next].baseUrl);
        test.reset();
    }

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (connection) {
            update.mutate(
                {
                    id: connection.id,
                    input: {
                        name,
                        baseUrl,
                        apiKey: removeKey
                            ? ""
                            : apiKey !== ""
                              ? apiKey
                              : undefined,
                    },
                },
                { onSuccess: onClose },
            );
        } else {
            create.mutate(
                { name, preset, baseUrl, apiKey },
                { onSuccess: onClose },
            );
        }
    }

    const keyMissing = info.keyRequired && apiKey === "" && !hasSavedKey;
    const canSubmit =
        name.trim() !== "" && baseUrl.trim() !== "" && !keyMissing;

    return (
        <Dialog
            open
            onClose={onClose}
            title={t(
                connection ? "ai.connection.editTitle" : "ai.connection.addTitle",
            )}
        >
            <form onSubmit={handleSubmit} className="flex flex-col gap-3">
                <SelectInput
                    label={t("ai.connection.preset")}
                    value={preset}
                    disabled={connection !== null}
                    onChange={(event) =>
                        choosePreset(event.target.value as AiPreset)
                    }
                    options={aiPresetOrder.map((p) => ({
                        value: p,
                        label: t(aiPresets[p].labelKey),
                    }))}
                />
                <TextInput
                    label={t("ai.connection.name")}
                    value={name}
                    maxLength={100}
                    required
                    onChange={(event) => {
                        setName(event.target.value);
                        setNameEdited(true);
                    }}
                />
                <TextInput
                    label={t("ai.connection.baseUrl")}
                    value={baseUrl}
                    required
                    inputMode="url"
                    onChange={(event) => {
                        setBaseUrl(event.target.value);
                        setUrlEdited(true);
                        test.reset();
                    }}
                />
                <div className="flex flex-col gap-1">
                    <TextInput
                        label={t("ai.connection.apiKey")}
                        type="password"
                        autoComplete="off"
                        value={apiKey}
                        disabled={removeKey}
                        placeholder={
                            hasSavedKey
                                ? t("ai.connection.apiKey.saved")
                                : info.keyRequired
                                  ? ""
                                  : t("ai.connection.apiKey.optional")
                        }
                        onChange={(event) => {
                            setApiKey(event.target.value);
                            test.reset();
                        }}
                    />
                    {connection?.hasKey && !info.keyRequired && (
                        <button
                            type="button"
                            onClick={() => {
                                setRemoveKey(!removeKey);
                                setApiKey("");
                            }}
                            className="cursor-pointer self-start text-caption text-fg-muted underline hover:text-fg"
                        >
                            {removeKey
                                ? t("ai.connection.apiKey.removed")
                                : t("ai.connection.apiKey.remove")}
                        </button>
                    )}
                </div>

                <div className="flex items-center gap-3">
                    <Button
                        size="lg"
                        loading={test.isPending}
                        disabled={baseUrl.trim() === "" || keyMissing}
                        onClick={() =>
                            test.mutate({ preset, baseUrl, apiKey })
                        }
                    >
                        {t(
                            test.isPending
                                ? "ai.connection.testing"
                                : "ai.connection.test",
                        )}
                    </Button>
                    {test.isSuccess && (
                        <p role="status" className="text-body-xs text-fg-muted">
                            ✓{" "}
                            {t("ai.connection.test.ok").replace(
                                "{n}",
                                String(test.data.length),
                            )}
                        </p>
                    )}
                    {test.isError && (
                        <FormError message={t(aiErrorKey(test.error))} />
                    )}
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
        </Dialog>
    );
}
