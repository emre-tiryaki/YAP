import { useId, useRef, type KeyboardEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { FiX } from "react-icons/fi";
import { useFocusTrap } from "../hooks/useFocusTrap";
import { useTranslate } from "../hooks/useTranslate";
import { Button } from "./Button";

type DialogProps = {
    open: boolean;
    onClose: () => void;
    title: string;
    size?: "sm" | "md" | "lg" | "xl";
    children: ReactNode;
    footer?: ReactNode;
    // Drops the body padding, for content that lays itself out edge to edge.
    bare?: boolean;
};

const widths = {
    sm: "max-w-sm",
    md: "max-w-md",
    lg: "max-w-2xl",
    xl: "max-w-4xl",
};

export function Dialog({
    open,
    onClose,
    title,
    size = "md",
    children,
    footer,
    bare,
}: DialogProps) {
    const t = useTranslate();
    const titleId = useId();
    const panelRef = useRef<HTMLDivElement>(null);
    useFocusTrap(panelRef, open);

    if (!open) return null;

    function handleKeyDown(event: KeyboardEvent) {
        if (event.key === "Escape") {
            event.stopPropagation();
            onClose();
        }
    }

    return createPortal(
        <div
            className="fixed inset-0 z-50 flex animate-fade-in items-center justify-center bg-backdrop p-4"
            onMouseDown={(event) => {
                if (event.target === event.currentTarget) onClose();
            }}
        >
            <div
                ref={panelRef}
                role="dialog"
                aria-modal="true"
                aria-labelledby={titleId}
                tabIndex={-1}
                onKeyDown={handleKeyDown}
                className={`app-opaque flex max-h-full w-full flex-col rounded-lg border border-line bg-surface-1 shadow-overlay outline-none ${widths[size]}`}
            >
                <div className="flex h-12 shrink-0 items-center justify-between border-b border-line px-4">
                    <h2 id={titleId} className="text-sm font-medium text-fg">
                        {title}
                    </h2>
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={onClose}
                        aria-label={t("common.close")}
                    >
                        <FiX aria-hidden />
                    </Button>
                </div>
                <div
                    className={`min-h-0 flex-1 overflow-auto ${bare ? "" : "p-4"}`}
                >
                    {children}
                </div>
                {footer && (
                    <div className="flex shrink-0 justify-end gap-2 border-t border-line px-4 py-3">
                        {footer}
                    </div>
                )}
            </div>
        </div>,
        document.body,
    );
}
