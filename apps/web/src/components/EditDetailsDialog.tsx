import { useState, type FormEvent } from "react";
import { useTranslate } from "../hooks/useTranslate";
import { FormError } from "./FormError";
import { Modal } from "./Modal";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

export type DetailsValue = {
    name: string;
    description: string;
};

type EditDetailsDialogProps = {
    isOpen: boolean;
    title: string;
    initial: DetailsValue;
    isPending: boolean;
    errorMessage: string | null;
    onSubmit: (value: DetailsValue) => void;
    onClose: () => void;
};

// Edits the name and description of a project or plan.
export function EditDetailsDialog({
    isOpen,
    title,
    onClose,
    ...formProps
}: EditDetailsDialogProps) {
    return (
        <Modal isOpen={isOpen} title={title} onClose={onClose}>
            <EditDetailsForm {...formProps} />
        </Modal>
    );
}

type EditDetailsFormProps = Omit<
    EditDetailsDialogProps,
    "isOpen" | "title" | "onClose"
>;

function EditDetailsForm({
    initial,
    isPending,
    errorMessage,
    onSubmit,
}: EditDetailsFormProps) {
    const t = useTranslate();
    const [name, setName] = useState(initial.name);
    const [description, setDescription] = useState(initial.description);

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        onSubmit({ name, description });
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={t("form.name")}
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
                autoFocus
            />
            <TextInput
                label={t("form.description")}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
            />
            <FormError message={errorMessage} />
            <div>
                <SubmitButton disabled={isPending || name.trim() === ""}>
                    {t("common.save")}
                </SubmitButton>
            </div>
        </form>
    );
}
