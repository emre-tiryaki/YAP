import { useEffect, type ReactNode } from "react";
import { useTranslate } from "../hooks/useTranslate";

type ModalProps = {
    isOpen: boolean;
    title: string;
    onClose: () => void;
    children: ReactNode;
};

export function Modal({ isOpen, title, onClose, children }: ModalProps) {
    const t = useTranslate();

    useEffect(() => {
        if (!isOpen) return;

        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") onClose();
        }

        document.addEventListener("keydown", handleKeyDown);
        return () => document.removeEventListener("keydown", handleKeyDown);
    }, [isOpen, onClose]);

    if (!isOpen) return null;

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
            onClick={onClose}
        >
            <div
                role="dialog"
                aria-modal="true"
                aria-label={title}
                className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
                onClick={(event) => event.stopPropagation()}
            >
                <div className="mb-4 flex items-center justify-between">
                    <h3 className="text-lg font-semibold">{title}</h3>
                    <button
                        type="button"
                        onClick={onClose}
                        aria-label={t("common.close")}
                        className="cursor-pointer rounded-md px-2 py-1 text-gray-500 hover:bg-gray-100"
                    >
                        ×
                    </button>
                </div>
                {children}
            </div>
        </div>
    );
}
