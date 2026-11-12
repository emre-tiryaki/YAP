import { useState, type FormEvent } from "react";
import { useTranslate } from "../hooks/useTranslate";
import { FormError } from "./FormError";
import { Modal } from "./Modal";
import { ProjectIconPicker } from "./ProjectIconPicker";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

export type DetailsValue = {
    name: string;
    description: string;
    // Only set when the dialog edits an icon (projects).
    icon?: string;
};

type EditDetailsDialogProps = {
    isOpen: boolean;
    title: string;
    initial: DetailsValue;
    // Shows the emoji picker and includes `icon` in the submitted value.
    withIcon?: boolean;
    isPending: boolean;
    errorMessage: string | null;
    onSubmit: (value: DetailsValue) => void;
    onClose: () => void;
};

// Edits the name and description of a project or plan, plus a project's icon.
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
    withIcon = false,
    isPending,
    errorMessage,
    onSubmit,
}: EditDetailsFormProps) {
    const t = useTranslate();
    const [name, setName] = useState(initial.name);
    const [description, setDescription] = useState(initial.description);
    const [icon, setIcon] = useState(initial.icon ?? "");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        onSubmit(withIcon ? { name, description, icon } : { name, description });
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
            {withIcon && <ProjectIconPicker value={icon} onChange={setIcon} />}
            <FormError message={errorMessage} />
            <div>
                <SubmitButton disabled={isPending || name.trim() === ""}>
                    {t("common.save")}
                </SubmitButton>
            </div>
        </form>
    );
}
