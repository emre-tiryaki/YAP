import { useState, type FormEvent } from "react";
import { useCreateProjects } from "../hooks/useProjects";
import { useTranslate } from "../hooks/useTranslate";
import { TextInput } from "./TextInput";
import { SubmitButton } from "./SubmitButton";

export function CreateProjectForm() {
    const t = useTranslate();
    const createProject = useCreateProjects();
    const [name, setName] = useState("");
    const [description, setDescription] = useState("");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        createProject.mutate(
            { name, description },
            {
                onSuccess: () => {
                    setName("");
                    setDescription("");
                },
            },
        );
    }

    return (
        <form
            onSubmit={handleSubmit}
            className="flex flex-col gap-3 rounded-lg border border-gray-200 p-4"
        >
            <h3 className="font-semibold">
                {t("workspace.projects.create.title")}
            </h3>
            <TextInput
                label={t("workspace.projects.create.name")}
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
            />
            <TextInput
                label={t("workspace.projects.create.description")}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
            />
            {createProject.isError && (
                <p className="text-sm text-red-600">{t("common.error")}</p>
            )}
            <div>
                <SubmitButton disabled={createProject.isPending}>
                    {t("workspace.projects.create.submit")}
                </SubmitButton>
            </div>
        </form>
    );
}
