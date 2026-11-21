import type { ReactNode } from "react";
import { Dialog } from "./Dialog";

type ModalProps = {
    isOpen: boolean;
    title: string;
    onClose: () => void;
    children: ReactNode;
};

// Kept for existing callers; new code uses Dialog directly.
export function Modal({ isOpen, title, onClose, children }: ModalProps) {
    return (
        <Dialog open={isOpen} onClose={onClose} title={title} size="md">
            {children}
        </Dialog>
    );
}
