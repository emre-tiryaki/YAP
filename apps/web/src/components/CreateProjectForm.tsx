import { ApiError } from "../api/client";
import { useState, type FormEvent } from "react";
import { useCreateProjects } from "../hooks/useProjects";
import { useTranslate } from "../hooks/useTranslate";
import { TextInput } from "./TextInput";
import { SubmitButton } from "./SubmitButton";
import { IconPicker } from "./IconPicker";
import { useNavigate } from "react-router";

type CreateProjectFormProps = {
    onDone: () => void;
};

export function CreateProjectForm({ onDone }: CreateProjectFormProps) {
    const t = useTranslate();
    const createProject = useCreateProjects();
    const navigate = useNavigate();
    const [name, setName] = useState("");
    const [description, setDescription] = useState("");
    const [icon, setIcon] = useState("");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        createProject.mutate(
            { name, description, icon },
            {
                onSuccess: (project) => {
                    onDone();
                    navigate(`/projects/${project.id}`);
                },
            },
        );
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={t("workspace.projects.create.name")}
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
                autoFocus
            />
            <TextInput
                label={t("workspace.projects.create.description")}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
            />
            <IconPicker value={icon} onChange={setIcon} />
            {createProject.isError && (
                <p className="text-sm text-red-600">
                    {createProject.error instanceof ApiError &&
                    createProject.error.message === "invalid_input"
                        ? t("project.invalid")
                        : t("common.error")}
                </p>
            )}
            <div>
                <SubmitButton disabled={createProject.isPending}>
                    {t("workspace.projects.create.submit")}
                </SubmitButton>
            </div>
        </form>
    );
}
