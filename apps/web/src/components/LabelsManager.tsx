import { useRef, useState, type FormEvent } from "react";
import { FiPlus, FiTrash2 } from "react-icons/fi";
import { ApiError } from "../api/client";
import { useCreateLabel, useDeleteLabel, useLabels, useUpdateLabel } from "../hooks/useLabels";
import { useTranslate } from "../hooks/useTranslate";
import type { Label, LabelColor } from "../types/label";
import { labelColors, labelDotClass } from "../util/labelColors";
import { Button } from "./Button";
import { ConfirmDialog } from "./ConfirmDialog";
import { EmptyState } from "./EmptyState";
import { FormError } from "./FormError";
import { InlineError } from "./InlineError";
import { Popover } from "./Popover";
import { Skeleton } from "./Skeleton";

type LabelsManagerProps = {
    projectId: string;
};

// The labels tab of a project: add, rename, recolour and delete labels.
export function LabelsManager({ projectId }: LabelsManagerProps) {
    const t = useTranslate();
    const labels = useLabels(projectId);
    const create = useCreateLabel(projectId);
    const [name, setName] = useState("");
    const [deleting, setDeleting] = useState<Label | null>(null);
    const remove = useDeleteLabel(projectId);

    function add(event: FormEvent) {
        event.preventDefault();
        const trimmed = name.trim();
        if (!trimmed) return;
        create.mutate({ name: trimmed, color: "gray" }, { onSuccess: () => setName("") });
    }

    const errorMessage = !create.error
        ? null
        : create.error instanceof ApiError && create.error.message === "conflict"
          ? t("label.manager.taken")
          : create.error instanceof ApiError && create.error.message === "invalid_input"
            ? t("label.manager.invalid")
            : t("common.error");

    return (
        <section aria-label={t("label.manager.title")} className="max-w-2xl">
            <form onSubmit={add} className="mb-4 flex items-center gap-2">
                <input
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    maxLength={50}
                    placeholder={t("label.manager.name")}
                    aria-label={t("label.manager.name")}
                    className="h-8 min-w-0 flex-1 rounded-md border border-line bg-layer-2 px-2 text-body-xs text-fg outline-none focus:border-accent"
                />
                <Button
                    type="submit"
                    variant="primary"
                    size="lg"
                    leadingIcon={FiPlus}
                    disabled={name.trim() === ""}
                    loading={create.isPending}
                >
                    {t("label.manager.add")}
                </Button>
            </form>
            <FormError message={errorMessage} />

            {labels.isLoading && <Skeleton className="h-24" />}
            {labels.isError && (
                <InlineError message={t("common.error")} onRetry={() => void labels.refetch()} />
            )}
            {labels.data?.length === 0 && (
                <EmptyState compact icon={FiPlus} title={t("label.manager.empty")} />
            )}
            {labels.data && labels.data.length > 0 && (
                <ul className="divide-y divide-line rounded-lg border border-line">
                    {labels.data.map((label) => (
                        <LabelRow
                            key={label.id}
                            label={label}
                            projectId={projectId}
                            onDelete={() => setDeleting(label)}
                        />
                    ))}
                </ul>
            )}

            <ConfirmDialog
                isOpen={deleting !== null}
                title={t("label.manager.delete")}
                message={`"${deleting?.name ?? ""}" — ${t("label.manager.deleteConfirm")}`}
                confirmLabel={t("common.delete")}
                isPending={remove.isPending}
                isError={remove.isError}
                onConfirm={() => {
                    if (deleting)
                        remove.mutate(deleting.id, { onSuccess: () => setDeleting(null) });
                }}
                onClose={() => {
                    setDeleting(null);
                    remove.reset();
                }}
            />
        </section>
    );
}

function LabelRow({
    label,
    projectId,
    onDelete,
}: {
    label: Label;
    projectId: string;
    onDelete: () => void;
}) {
    const t = useTranslate();
    const update = useUpdateLabel(projectId);
    const [name, setName] = useState(label.name);
    const [isPicking, setIsPicking] = useState(false);
    const swatchRef = useRef<HTMLButtonElement>(null);

    function saveName() {
        const next = name.trim();
        if (next === "" || next === label.name) {
            setName(label.name);
            return;
        }
        update.mutate(
            { labelId: label.id, input: { name: next } },
            { onError: () => setName(label.name) },
        );
    }

    function setColor(color: LabelColor) {
        setIsPicking(false);
        if (color !== label.color)
            update.mutate({ labelId: label.id, input: { color } });
    }

    return (
        <li className="flex items-center gap-3 px-3 py-2">
            <button
                ref={swatchRef}
                type="button"
                onClick={() => setIsPicking(!isPicking)}
                aria-label={`${t("label.manager.recolor")}: ${label.name}`}
                aria-haspopup="menu"
                aria-expanded={isPicking}
                className="flex size-6 shrink-0 cursor-pointer items-center justify-center rounded-md hover:bg-layer-1-hover"
            >
                <span className={`size-3 rounded-full ${labelDotClass(label.color)}`} />
            </button>
            <Popover open={isPicking} onClose={() => setIsPicking(false)} anchorRef={swatchRef}>
                <div role="menu" className="grid grid-cols-5 gap-1 p-1">
                    {labelColors.map(({ value, dot }) => (
                        <button
                            key={value}
                            type="button"
                            role="menuitemradio"
                            aria-checked={value === label.color}
                            aria-label={t(`label.color.${value}`)}
                            title={t(`label.color.${value}`)}
                            onClick={() => setColor(value)}
                            className="flex size-7 cursor-pointer items-center justify-center rounded-md hover:bg-layer-1-hover aria-checked:bg-layer-3"
                        >
                            <span className={`size-3.5 rounded-full ${dot}`} />
                        </button>
                    ))}
                </div>
            </Popover>
            <input
                value={name}
                maxLength={50}
                onChange={(event) => setName(event.target.value)}
                onBlur={saveName}
                onKeyDown={(event) => {
                    if (event.key === "Enter") event.currentTarget.blur();
                    if (event.key === "Escape") {
                        setName(label.name);
                        event.currentTarget.blur();
                    }
                }}
                aria-label={`${t("label.manager.rename")}: ${label.name}`}
                className="h-7 min-w-0 flex-1 rounded-md border border-transparent bg-transparent px-2 text-body-xs text-fg hover:border-line focus:border-accent focus:outline-none"
            />
            <Button
                variant="ghost"
                size="sm"
                aria-label={`${t("label.manager.delete")}: ${label.name}`}
                onClick={onDelete}
            >
                <FiTrash2 aria-hidden />
            </Button>
        </li>
    );
}
