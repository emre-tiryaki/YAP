import { useState, type FormEvent } from "react";
import { ApiError } from "../api/client";
import { useCreateStatus } from "../hooks/usePlans";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { StatusCategory } from "../types/task";
import { statusColors } from "../util/statusColors";
import { SelectInput } from "./SelectInput";
import { StatusColorPicker } from "./StatusColorPicker";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

type CreateStatusFormProps = {
    projectId: string;
    planId: string;
    onDone: () => void;
};

const categories: StatusCategory[] = ["todo", "in_progress", "done"];

function errorKey(error: Error): TranslationKey {
    if (!(error instanceof ApiError)) return "common.error";
    switch (error.message) {
        case "not_found":
            return "plan.status.create.notFound";
        case "invalid_input":
            return "plan.status.create.invalid";
        default:
            return "common.error";
    }
}

export function CreateStatusForm({
    projectId,
    planId,
    onDone,
}: CreateStatusFormProps) {
    const t = useTranslate();
    const createStatus = useCreateStatus(projectId, planId);
    const [title, setTitle] = useState("");
    const [color, setColor] = useState(statusColors[0]);
    const [category, setCategory] = useState<StatusCategory>("in_progress");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        createStatus.mutate({ title, color, category }, { onSuccess: onDone });
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={t("plan.status.create.name")}
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                required
                autoFocus
            />
            <StatusColorPicker value={color} onChange={setColor} />
            <SelectInput
                label={t("plan.status.create.category")}
                value={category}
                onChange={(event) => {
                    const next = categories.find(
                        (c) => c === event.target.value,
                    );
                    if (next) setCategory(next);
                }}
                options={categories.map((c) => ({
                    value: c,
                    label: t(`status.category.${c}`),
                }))}
            />
            <p className="text-xs text-gray-500">
                {t("plan.status.create.doneHint")}
            </p>
            {createStatus.error && (
                <p role="alert" className="text-sm text-red-600">
                    {t(errorKey(createStatus.error))}
                </p>
            )}
            <div>
                <SubmitButton
                    disabled={createStatus.isPending || title.trim() === ""}
                >
                    {t("plan.status.create.submit")}
                </SubmitButton>
            </div>
        </form>
    );
}
