import { useState, type FormEvent } from "react";
import { ApiError } from "../api/client";
import { useCreateStatus } from "../hooks/usePlans";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { StatusCategory } from "../types/task";
import { SelectInput } from "./SelectInput";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

type CreateStatusFormProps = {
    projectId: string;
    planId: string;
    onDone: () => void;
};

const colors = [
    "#9ca3af",
    "#3b82f6",
    "#a855f7",
    "#ec4899",
    "#ef4444",
    "#f97316",
    "#eab308",
    "#22c55e",
];

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
    const [color, setColor] = useState(colors[0]);
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
            <fieldset className="flex flex-col gap-1 text-sm text-gray-700">
                <legend className="mb-1">{t("plan.status.create.color")}</legend>
                <div className="flex flex-wrap gap-2">
                    {colors.map((c) => (
                        <label
                            key={c}
                            className="cursor-pointer rounded-full has-checked:ring-2 has-checked:ring-gray-900 has-checked:ring-offset-2 has-focus-visible:ring-2 has-focus-visible:ring-gray-500"
                        >
                            <input
                                type="radio"
                                name="status-color"
                                value={c}
                                checked={color === c}
                                onChange={() => setColor(c)}
                                aria-label={c}
                                className="sr-only"
                            />
                            <span
                                aria-hidden
                                className="block h-6 w-6 rounded-full"
                                style={{ backgroundColor: c }}
                            />
                        </label>
                    ))}
                </div>
            </fieldset>
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
