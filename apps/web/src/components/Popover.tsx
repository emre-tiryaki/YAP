import {
    useLayoutEffect,
    useRef,
    type ReactNode,
    type RefObject,
} from "react";
import { createPortal } from "react-dom";
import { useDismiss } from "../hooks/useDismiss";

type PopoverProps = {
    open: boolean;
    onClose: () => void;
    anchorRef: RefObject<HTMLElement | null>;
    placement?: "bottom-start" | "bottom-end" | "right-start";
    children: ReactNode;
    className?: string;
};

const GAP = 4;
const MARGIN = 8;

export function Popover({
    open,
    onClose,
    anchorRef,
    placement = "bottom-start",
    children,
    className = "",
}: PopoverProps) {
    const ref = useRef<HTMLDivElement>(null);
    useDismiss(ref, open, onClose, [anchorRef]);

    useLayoutEffect(() => {
        if (!open) return;

        function position() {
            const anchor = anchorRef.current;
            const element = ref.current;
            if (!anchor || !element) return;
            const rect = anchor.getBoundingClientRect();
            const width = element.offsetWidth;
            const height = element.offsetHeight;

            let left =
                placement === "bottom-end"
                    ? rect.right - width
                    : placement === "right-start"
                      ? rect.right + GAP
                      : rect.left;
            let top =
                placement === "right-start" ? rect.top : rect.bottom + GAP;

            // Flip above when there is no room below.
            if (
                top + height > window.innerHeight - MARGIN &&
                rect.top - height - GAP > MARGIN &&
                placement !== "right-start"
            )
                top = rect.top - height - GAP;
            top = Math.max(
                MARGIN,
                Math.min(top, window.innerHeight - height - MARGIN),
            );
            left = Math.max(
                MARGIN,
                Math.min(left, window.innerWidth - width - MARGIN),
            );
            element.style.top = `${top}px`;
            element.style.left = `${left}px`;
        }

        position();
        window.addEventListener("resize", position);
        window.addEventListener("scroll", position, true);
        return () => {
            window.removeEventListener("resize", position);
            window.removeEventListener("scroll", position, true);
        };
    }, [open, anchorRef, placement]);

    if (!open) return null;

    return createPortal(
        <div
            ref={ref}
            className={`app-opaque fixed z-50 min-w-48 max-w-100 rounded-lg border border-line bg-surface-1 p-1 shadow-overlay ${className}`}
        >
            {children}
        </div>,
        document.body,
    );
}
