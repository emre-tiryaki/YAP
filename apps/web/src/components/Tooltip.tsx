import {
    cloneElement,
    useEffect,
    useId,
    useState,
    type FocusEvent,
    type MouseEvent,
    type ReactElement,
} from "react";
import { createPortal } from "react-dom";
import { Kbd } from "./Kbd";

type TooltipProps = {
    label: string;
    shortcut?: string[];
    children: ReactElement<Record<string, unknown>>;
    side?: "top" | "bottom" | "right";
};

const HOVER_DELAY = 400;

function position(side: "top" | "bottom" | "right", rect: DOMRect) {
    if (side === "right")
        return { top: rect.top + rect.height / 2, left: rect.right + 8 };
    if (side === "top")
        return { top: rect.top - 8, left: rect.left + rect.width / 2 };
    return { top: rect.bottom + 8, left: rect.left + rect.width / 2 };
}


// Shows `label` after a short hover, or at once on keyboard focus. Rendered in
// a portal with fixed position so scrolling containers cannot clip it.
export function Tooltip({
    label,
    shortcut,
    children,
    side = "bottom",
}: TooltipProps) {
    const id = useId();
    const [pos, setPos] = useState<{ top: number; left: number } | null>(null);
    // The element being hovered; the tooltip opens once it stays hovered.
    const [hovered, setHovered] = useState<HTMLElement | null>(null);

    useEffect(() => {
        if (!hovered) return;
        const id = window.setTimeout(() => {
            const rect = hovered.getBoundingClientRect();
            setPos(position(side, rect));
        }, HOVER_DELAY);
        return () => window.clearTimeout(id);
    }, [hovered, side]);

    function hide() {
        setHovered(null);
        setPos(null);
    }

    const handlers = {
        onMouseEnter: (event: MouseEvent<HTMLElement>) =>
            setHovered(event.currentTarget),
        onMouseLeave: hide,
        onFocus: (event: FocusEvent<HTMLElement>) => {
            if (event.currentTarget.matches(":focus-visible"))
                setPos(position(side, event.currentTarget.getBoundingClientRect()));
        },
        onBlur: hide,
    };

    const transform =
        side === "right"
            ? "-translate-y-1/2"
            : side === "top"
              ? "-translate-x-1/2 -translate-y-full"
              : "-translate-x-1/2";

    return (
        <>
            {cloneElement(children, {
                ...handlers,
                "aria-describedby": pos ? id : undefined,
            })}
            {pos &&
                createPortal(
                    <span
                        id={id}
                        role="tooltip"
                        style={{ top: pos.top, left: pos.left }}
                        className={`pointer-events-none fixed z-60 flex items-center gap-1.5 rounded-md bg-gray-900 px-2 py-1 text-caption whitespace-nowrap text-white ${transform}`}
                    >
                        {label}
                        {shortcut && <Kbd keys={shortcut} />}
                    </span>,
                    document.body,
                )}
        </>
    );
}
