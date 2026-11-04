import { useState, type FormEvent } from "react";
import { useCreateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";
import { TextInput } from "./TextInput";
import { SelectInput } from "./SelectInput";
import { SubmitButton } from "./SubmitButton";

type CreateTaskFormProps = {
    projects: Project[];
};

export function CreateTaskForm({ projects }: CreateTaskFormProps) {
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
            {
                onSuccess: () => {
                    setTitle("");
                    setDueDate("");
                },
            },
        );
    }

    return (
        <form
            onSubmit={handleSubmit}
            className="flex flex-col gap-3 rounded-lg border border-gray-200 p-4"
        >
            <h3>{t("workspace.tasks.create.title")}</h3>
            <TextInput
                label={t("workspace.tasks.create.name")}
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                required
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
            <TextInput
                label={t("workspace.tasks.create.dueDate")}
                type="date"
                value={dueDate}
                onChange={(event) => setDueDate(event.target.value)}
            />
            {createTask.isError && <p className="text-sm text-red-600">{t('common.error')}</p>}
            <div>
                <SubmitButton disabled={createTask.isPending || selectedProjectId === ''}>
                    {t('workspace.tasks.create.submit')}
                </SubmitButton>
            </div>
        </form>
    );
}
