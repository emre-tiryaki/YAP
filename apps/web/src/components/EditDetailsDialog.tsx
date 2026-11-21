import { useState, type FormEvent } from "react";
import { useTranslate } from "../hooks/useTranslate";
import { FormError } from "./FormError";
import { Modal } from "./Modal";
import { IconPicker } from "./IconPicker";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

export type DetailsValue = {
    name: string;
    description: string;
    // Only set when the dialog edits an icon.
    icon?: string;
    // Only set when the dialog edits a project identifier.
    identifier?: string;
};

type EditDetailsDialogProps = {
    isOpen: boolean;
    title: string;
    initial: DetailsValue;
    // Shows the emoji picker and includes `icon` in the submitted value.
    withIcon?: boolean;
    // Shows the project identifier field (ABC) and includes `identifier`.
    withIdentifier?: boolean;
    isPending: boolean;
    errorMessage: string | null;
    onSubmit: (value: DetailsValue) => void;
    onClose: () => void;
};

// Edits the name, description and icon of a project or plan.
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
    withIdentifier = false,
    isPending,
    errorMessage,
    onSubmit,
}: EditDetailsFormProps) {
    const t = useTranslate();
    const [name, setName] = useState(initial.name);
    const [description, setDescription] = useState(initial.description);
    const [icon, setIcon] = useState(initial.icon ?? "");
    const [identifier, setIdentifier] = useState(initial.identifier ?? "");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        onSubmit({
            name,
            description,
            ...(withIcon ? { icon } : {}),
            ...(withIdentifier ? { identifier } : {}),
        });
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
            {withIdentifier && (
                <div className="flex flex-col gap-1">
                    <TextInput
                        label={t("project.identifier")}
                        value={identifier}
                        onChange={(event) =>
                            setIdentifier(event.target.value.toUpperCase())
                        }
                        maxLength={5}
                        required
                    />
                    <p className="text-caption text-fg-subtle">
                        {t("project.identifier.hint")}
                    </p>
                </div>
            )}
            {withIcon && <IconPicker value={icon} onChange={setIcon} />}
            <FormError message={errorMessage} />
            <div>
                <SubmitButton disabled={isPending || name.trim() === ""}>
                    {t("common.save")}
                </SubmitButton>
            </div>
        </form>
    );
}
