import { useRef, useState } from "react";
import { FiChevronDown } from "react-icons/fi";
import { useAiConnections, useAiModels } from "../hooks/useAi";
import type { AiModelChoice } from "../hooks/useAiModelChoice";
import { openSettings } from "../hooks/useSettingsDialog";
import { useTranslate } from "../hooks/useTranslate";
import { Button } from "./Button";
import { Popover } from "./Popover";
import { SelectInput } from "./SelectInput";

type AiModelSelectProps = {
    value: AiModelChoice | null;
    onChange: (choice: AiModelChoice) => void;
    disabled?: boolean;
};

const rowClass =
    "flex h-8 w-full cursor-pointer items-center rounded-md px-2 text-left text-body-xs hover:bg-layer-1-hover";

// The model of the next message: a connection, then one of its models (or a
// name typed by hand, for servers that do not list their models).
export function AiModelSelect({ value, onChange, disabled }: AiModelSelectProps) {
    const t = useTranslate();
    const anchorRef = useRef<HTMLButtonElement>(null);
    const [open, setOpen] = useState(false);
    const connections = useAiConnections();
    const [connectionId, setConnectionId] = useState(value?.connectionId ?? "");
    const [filter, setFilter] = useState("");
    const [custom, setCustom] = useState("");

    const list = connections.data ?? [];
    const activeId =
        list.find((c) => c.id === connectionId)?.id ?? list[0]?.id ?? "";
    const models = useAiModels(activeId, open);
    const shown = (models.data ?? []).filter((m) =>
        m.toLowerCase().includes(filter.trim().toLowerCase()),
    );
    const current = list.find((c) => c.id === value?.connectionId);

    function pick(model: string) {
        if (!activeId || !model) return;
        onChange({ connectionId: activeId, model });
        setOpen(false);
    }

    return (
        <>
            <Button
                ref={anchorRef}
                size="md"
                disabled={disabled}
                aria-haspopup="dialog"
                aria-expanded={open}
                aria-label={t("assistant.model.select")}
                onClick={() => {
                    setConnectionId(value?.connectionId ?? "");
                    setOpen(!open);
                }}
            >
                <span className="max-w-48 truncate">
                    {value
                        ? `${value.model}${current ? "" : ` (${t("assistant.model.removed")})`}`
                        : t("assistant.model.none")}
                </span>
                <FiChevronDown aria-hidden />
            </Button>
            <Popover
                open={open}
                onClose={() => setOpen(false)}
                anchorRef={anchorRef}
                className="flex w-80 flex-col gap-2 rounded-lg border border-line bg-surface-1 p-3 shadow-lg"
            >
                <SelectInput
                    label={t("settings.ai.connections")}
                    value={activeId}
                    onChange={(event) => {
                        setConnectionId(event.target.value);
                        setFilter("");
                    }}
                    options={list.map((c) => ({ value: c.id, label: c.name }))}
                />
                <input
                    aria-label={t("assistant.model.filter")}
                    placeholder={t("assistant.model.filter")}
                    value={filter}
                    onChange={(event) => setFilter(event.target.value)}
                    className="rounded-md border border-line bg-layer-2 px-2 py-1.5 text-body-xs"
                />
                <div role="listbox" className="max-h-56 overflow-y-auto">
                    {models.isPending && activeId !== "" ? (
                        <p className="px-2 py-1 text-body-xs text-fg-subtle">
                            {t("assistant.model.loading")}
                        </p>
                    ) : models.isError ? (
                        <p role="alert" className="px-2 py-1 text-body-xs text-danger">
                            {t("ai.error.unreachable")}
                        </p>
                    ) : shown.length === 0 ? (
                        <p className="px-2 py-1 text-body-xs text-fg-subtle">
                            {t("assistant.model.empty")}
                        </p>
                    ) : (
                        shown.map((model) => (
                            <button
                                key={model}
                                type="button"
                                role="option"
                                aria-selected={
                                    value?.connectionId === activeId &&
                                    value.model === model
                                }
                                className={`${rowClass} ${
                                    value?.connectionId === activeId &&
                                    value.model === model
                                        ? "bg-layer-1-hover font-medium"
                                        : ""
                                }`}
                                onClick={() => pick(model)}
                            >
                                <span className="truncate">{model}</span>
                            </button>
                        ))
                    )}
                </div>
                <form
                    className="flex gap-2"
                    onSubmit={(event) => {
                        event.preventDefault();
                        pick(custom.trim());
                        setCustom("");
                    }}
                >
                    <input
                        aria-label={t("assistant.model.custom")}
                        placeholder={t("assistant.model.custom.placeholder")}
                        value={custom}
                        onChange={(event) => setCustom(event.target.value)}
                        className="min-w-0 flex-1 rounded-md border border-line bg-layer-2 px-2 py-1.5 text-body-xs"
                    />
                    <Button size="md" type="submit" disabled={custom.trim() === ""}>
                        {t("assistant.model.custom.use")}
                    </Button>
                </form>
                <button
                    type="button"
                    onClick={() => {
                        setOpen(false);
                        openSettings("ai");
                    }}
                    className="cursor-pointer self-start text-caption text-fg-muted underline hover:text-fg"
                >
                    {t("assistant.model.manage")}
                </button>
            </Popover>
        </>
    );
}
