import { useState, type FormEvent } from "react";
import { FiTrash2 } from "react-icons/fi";
import { ApiError } from "../api/client";
import { useUpdateStatus } from "../hooks/usePlans";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import { FormError } from "./FormError";
import { StatusColorPicker } from "./StatusColorPicker";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

type EditStatusFormProps = {
    projectId: string;
    planId: string;
    status: Status;
    onDone: () => void;
    onDelete: () => void;
};

function errorKey(error: Error): TranslationKey {
    if (!(error instanceof ApiError)) return "common.error";
    switch (error.message) {
        case "not_found":
            return "plan.status.edit.notFound";
        case "invalid_input":
            return "plan.status.create.invalid";
        default:
            return "common.error";
    }
}

export function EditStatusForm({
    projectId,
    planId,
    status,
    onDone,
    onDelete,
}: EditStatusFormProps) {
    const t = useTranslate();
    const updateStatus = useUpdateStatus(projectId, planId);
    const [title, setTitle] = useState(status.title);
    const [description, setDescription] = useState(status.description);
    const [color, setColor] = useState(status.color);

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        updateStatus.mutate(
            { statusId: status.id, input: { title, description, color } },
            { onSuccess: onDone },
        );
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={t("plan.status.create.name")}
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                required
                autoFocus
                onFocus={(event) => event.target.select()}
            />
            <TextInput
                label={t("plan.status.create.description")}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
            />
            <StatusColorPicker value={color} onChange={setColor} />
            <FormError
                message={
                    updateStatus.error ? t(errorKey(updateStatus.error)) : null
                }
            />
            <div className="flex items-center justify-between">
                <SubmitButton
                    disabled={updateStatus.isPending || title.trim() === ""}
                >
                    {t("common.save")}
                </SubmitButton>
                <button
                    type="button"
                    onClick={onDelete}
                    className="flex cursor-pointer items-center gap-1 rounded-md px-3 py-2 text-sm text-red-600 hover:bg-red-50"
                >
                    <FiTrash2 aria-hidden />
                    {t("plan.status.delete.action")}
                </button>
            </div>
        </form>
    );
}
