import { useTranslate } from "../hooks/useTranslate";
import { Modal } from "./Modal";

type ConfirmDialogProps = {
    isOpen: boolean;
    title: string;
    message: string;
    confirmLabel: string;
    isPending?: boolean;
    isError?: boolean;
    onConfirm: () => void;
    onClose: () => void;
};

export function ConfirmDialog({
    isOpen,
    title,
    message,
    confirmLabel,
    isPending = false,
    isError = false,
    onConfirm,
    onClose,
}: ConfirmDialogProps) {
    const t = useTranslate();

    return (
        <Modal isOpen={isOpen} title={title} onClose={onClose}>
            <p className="text-sm text-gray-600">{message}</p>
            {isError && (
                <p role="alert" className="mt-3 text-sm text-red-600">
                    {t("common.error")}
                </p>
            )}
            <div className="mt-6 flex justify-end gap-2">
                <button
                    type="button"
                    onClick={onClose}
                    className="cursor-pointer rounded-md border border-gray-300 px-4 py-2 text-sm hover:bg-gray-100"
                >
                    {t("common.cancel")}
                </button>
                <button
                    type="button"
                    onClick={onConfirm}
                    disabled={isPending}
                    autoFocus
                    className="cursor-pointer rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
                >
                    {confirmLabel}
                </button>
            </div>
        </Modal>
    );
}
