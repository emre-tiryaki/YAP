import { useState, type FormEvent } from "react";
import { ApiError } from "../api/client";
import { useCreateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import { SelectInput } from "./SelectInput";
import { DatePicker } from "./DatePicker";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

type CreatePlanTaskFormProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    initialStatusId: string;
    onDone: () => void;
};

export function CreatePlanTaskForm({
    projectId,
    planId,
    statuses,
    initialStatusId,
    onDone,
}: CreatePlanTaskFormProps) {
    const t = useTranslate();
    const createTask = useCreateTask();
    const [title, setTitle] = useState("");
    const [description, setDescription] = useState("");
    const [statusId, setStatusId] = useState(initialStatusId);
    const [dueDate, setDueDate] = useState("");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        createTask.mutate(
            {
                projectId,
                input: {
                    title,
                    description,
                    planId,
                    statusId,
                    dueDate: dueDate ? new Date(`${dueDate}T00:00:00`) : null,
                },
            },
            { onSuccess: onDone },
        );
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={t("plan.task.create.name")}
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                required
                autoFocus
            />
            <TextInput
                label={t("plan.task.create.description")}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
            />
            <SelectInput
                label={t("plan.task.create.status")}
                value={statusId}
                onChange={(event) => setStatusId(event.target.value)}
                options={statuses.map((status) => ({
                    value: status.id,
                    label: status.title,
                }))}
            />
            <DatePicker
                label={t("plan.task.create.dueDate")}
                showLabel
                value={dueDate}
                onChange={setDueDate}
            />
            {createTask.error && (
                <p role="alert" className="text-sm text-red-600">
                    {createTask.error instanceof ApiError &&
                    createTask.error.message === "invalid_input"
                        ? t("plan.task.create.invalid")
                        : t("common.error")}
                </p>
            )}
            <div>
                <SubmitButton
                    disabled={createTask.isPending || title.trim() === ""}
                >
                    {t("plan.task.create.submit")}
                </SubmitButton>
            </div>
        </form>
    );
}
