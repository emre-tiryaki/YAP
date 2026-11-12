import { useState, type FormEvent } from "react";
import { useCreateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";
import { TextInput } from "./TextInput";
import { SelectInput } from "./SelectInput";
import { DatePicker } from "./DatePicker";
import { SubmitButton } from "./SubmitButton";

type CreateTaskFormProps = {
    projects: Project[];
    onDone: () => void;
};

export function CreateTaskForm({ projects, onDone }: CreateTaskFormProps) {
    const t = useTranslate();
    const createTask = useCreateTask();
    const [title, setTitle] = useState("");
    const [projectId, setProjectId] = useState("");
    const [dueDate, setDueDate] = useState("");

    const selectedProjectId = projectId || (projects[0]?.id ?? "");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        createTask.mutate(
            {
                projectId: selectedProjectId,
                input: {
                    title,
                    dueDate: dueDate ? new Date(`${dueDate}T00:00:00`) : null,
                },
            },
            { onSuccess: onDone },
        );
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={t("workspace.tasks.create.name")}
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                required
                autoFocus
            />
            <SelectInput
                label={t("workspace.tasks.create.project")}
                value={selectedProjectId}
                onChange={(event) => setProjectId(event.target.value)}
                options={projects.map((project) => ({
                    value: project.id,
                    label: project.name,
                }))}
            />
            <DatePicker
                label={t("workspace.tasks.create.dueDate")}
                showLabel
                value={dueDate}
                onChange={setDueDate}
            />
            {createTask.isError && (
                <p className="text-sm text-red-600">{t("common.error")}</p>
            )}
            <div>
                <SubmitButton
                    disabled={createTask.isPending || selectedProjectId === ""}
                >
                    {t("workspace.tasks.create.submit")}
                </SubmitButton>
            </div>
        </form>
    );
}
