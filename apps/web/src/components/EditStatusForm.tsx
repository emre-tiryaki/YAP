import { useState, type FormEvent } from "react";
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
}: EditStatusFormProps) {
    const t = useTranslate();
    const updateStatus = useUpdateStatus(projectId, planId);
    const [title, setTitle] = useState(status.title);
    const [color, setColor] = useState(status.color);

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        updateStatus.mutate(
            { statusId: status.id, input: { title, color } },
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
            <StatusColorPicker value={color} onChange={setColor} />
            <FormError
                message={
                    updateStatus.error ? t(errorKey(updateStatus.error)) : null
                }
            />
            <div>
                <SubmitButton
                    disabled={updateStatus.isPending || title.trim() === ""}
                >
                    {t("common.save")}
                </SubmitButton>
            </div>
        </form>
    );
}
