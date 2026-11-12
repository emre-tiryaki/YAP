import { useState, type FormEvent } from "react";
import { useTranslate } from "../hooks/useTranslate";
import type { UpdateTaskInput } from "../types/task";
import { fromDateInputValue, toDateInputValue } from "../util/date";
import { FormError } from "./FormError";
import { Modal } from "./Modal";
import { DatePicker } from "./DatePicker";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

export type EditableTask = {
    title: string;
    // Undefined when the caller does not know the description; the field is
    // then hidden and left unchanged.
    description?: string;
    dueDate: Date | null;
};

type EditTaskDialogProps = {
    isOpen: boolean;
    task: EditableTask;
    isPending: boolean;
    errorMessage: string | null;
    onSubmit: (input: UpdateTaskInput) => void;
    onClose: () => void;
};

export function EditTaskDialog({
    isOpen,
    onClose,
    ...formProps
}: EditTaskDialogProps) {
    const t = useTranslate();
    return (
        <Modal isOpen={isOpen} title={t("task.edit.title")} onClose={onClose}>
            <EditTaskForm {...formProps} />
        </Modal>
    );
}

type EditTaskFormProps = Omit<EditTaskDialogProps, "isOpen" | "onClose">;

function EditTaskForm({
    task,
    isPending,
    errorMessage,
    onSubmit,
}: EditTaskFormProps) {
    const t = useTranslate();
    const [title, setTitle] = useState(task.title);
    const [description, setDescription] = useState(task.description ?? "");
    const [dueDate, setDueDate] = useState(toDateInputValue(task.dueDate));

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        onSubmit({
            title,
            ...(task.description !== undefined && { description }),
            dueDate: fromDateInputValue(dueDate),
        });
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={t("form.title")}
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                required
                autoFocus
            />
            {task.description !== undefined && (
                <TextInput
                    label={t("form.description")}
                    value={description}
                    onChange={(event) => setDescription(event.target.value)}
                />
            )}
            <DatePicker
                label={t("form.dueDate")}
                showLabel
                value={dueDate}
                onChange={setDueDate}
            />
            <FormError message={errorMessage} />
            <div>
                <SubmitButton disabled={isPending || title.trim() === ""}>
                    {t("common.save")}
                </SubmitButton>
            </div>
        </form>
    );
}
