import { useState, type FormEvent } from "react";
import { useTranslate } from "../hooks/useTranslate";
import { FormError } from "./FormError";
import { Modal } from "./Modal";
import { SubmitButton } from "./SubmitButton";
import { TextInput } from "./TextInput";

type RenameDialogProps = {
    isOpen: boolean;
    title: string;
    label: string;
    initialName: string;
    isPending: boolean;
    errorMessage: string | null;
    onSubmit: (name: string) => void;
    onClose: () => void;
};

export function RenameDialog({
    isOpen,
    title,
    onClose,
    ...formProps
}: RenameDialogProps) {
    // The form only mounts while open, so it starts from the current name.
    return (
        <Modal isOpen={isOpen} title={title} onClose={onClose}>
            <RenameForm {...formProps} />
        </Modal>
    );
}

type RenameFormProps = Omit<RenameDialogProps, "isOpen" | "title" | "onClose">;

function RenameForm({
    label,
    initialName,
    isPending,
    errorMessage,
    onSubmit,
}: RenameFormProps) {
    const t = useTranslate();
    const [name, setName] = useState(initialName);

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        onSubmit(name);
    }

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <TextInput
                label={label}
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
                autoFocus
                onFocus={(event) => event.target.select()}
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
