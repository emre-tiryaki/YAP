import { useState, type FormEvent } from "react";
import { ApiError } from "../api/client";
import { useCreatePlan } from "../hooks/useProjects";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

type CreatePlanFormProps = {
    projectId: string;
    onDone: () => void;
};

function errorKey(error: Error): TranslationKey {
    if (!(error instanceof ApiError)) return "common.error";
    switch (error.message) {
        case "conflict":
            return "project.plans.create.duplicate";
        case "not_found":
            return "project.plans.create.notFound";
        case "invalid_input":
            return "project.plans.create.invalid";
        default:
            return "common.error";
    }
}

export function CreatePlanForm({ projectId, onDone }: CreatePlanFormProps) {
    const t = useTranslate();
    const createPlan = useCreatePlan(projectId);
    const [name, setName] = useState("");
    const [description, setDescription] = useState("");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        createPlan.mutate({ name, description }, { onSuccess: onDone });
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={t("project.plans.create.name")}
                value={name}
                onChange={(event) => {
                    setName(event.target.value);
                    if (createPlan.isError) createPlan.reset();
                }}
                required
                autoFocus
            />
            <TextInput
                label={t("project.plans.create.description")}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
            />
            {createPlan.error && (
                <p role="alert" className="text-sm text-red-600">
                    {t(errorKey(createPlan.error))}
                </p>
            )}
            <div>
                <SubmitButton
                    disabled={createPlan.isPending || name.trim() === ""}
                >
                    {t("project.plans.create.submit")}
                </SubmitButton>
            </div>
        </form>
    );
}
