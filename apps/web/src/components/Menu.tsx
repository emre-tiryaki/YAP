import {
    useEffect,
    useRef,
    useState,
    type KeyboardEvent,
    type ReactNode,
    type RefObject,
} from "react";
import { MenuContext } from "../hooks/useMenu";
import { Popover } from "./Popover";

type TriggerProps = {
    ref: RefObject<HTMLButtonElement | null>;
    onClick: () => void;
    onKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
    "aria-haspopup": "menu";
    "aria-expanded": boolean;
};

type MenuProps = {
    trigger: (props: TriggerProps) => ReactNode;
    children: ReactNode;
    placement?: "bottom-start" | "bottom-end" | "right-start";
};

// WAI-ARIA menu button: arrows move, Home/End jump, typeahead by first letter,
// Esc closes and returns focus to the trigger.
export function Menu({ trigger, children, placement = "bottom-start" }: MenuProps) {
    const [open, setOpen] = useState(false);
    const [focusLast, setFocusLast] = useState(false);
    const triggerRef = useRef<HTMLButtonElement>(null);
    const listRef = useRef<HTMLDivElement | null>(null);

    function close() {
        // Only take focus back when it was inside the list; an outside click
        // has moved it somewhere else on purpose.
        const hadFocus = listRef.current?.contains(document.activeElement);
        setOpen(false);
        if (hadFocus) triggerRef.current?.focus();
    }

    function items() {
        return Array.from(
            listRef.current?.querySelectorAll<HTMLElement>(
                '[role="menuitem"]:not(:disabled)',
            ) ?? [],
        );
    }

    // Focuses the first (or last) item once the list is mounted.
    useEffect(() => {
        if (!open) return;
        const all = Array.from(
            listRef.current?.querySelectorAll<HTMLElement>(
                '[role="menuitem"]:not(:disabled)',
            ) ?? [],
        );
        (focusLast ? all[all.length - 1] : all[0])?.focus();
    }, [open, focusLast]);

    function handleListKeyDown(event: KeyboardEvent<HTMLDivElement>) {
        const all = items();
        const index = all.indexOf(document.activeElement as HTMLElement);
        let next: HTMLElement | undefined;
        if (event.key === "ArrowDown") next = all[(index + 1) % all.length];
        else if (event.key === "ArrowUp")
            next = all[(index - 1 + all.length) % all.length];
        else if (event.key === "Home") next = all[0];
        else if (event.key === "End") next = all[all.length - 1];
        else if (event.key === "Tab") {
            event.preventDefault();
            return;
        } else if (event.key.length === 1 && /\S/.test(event.key)) {
            const letter = event.key.toLowerCase();
            const ordered = [...all.slice(index + 1), ...all.slice(0, index + 1)];
            next = ordered.find((item) =>
                item.textContent?.trim().toLowerCase().startsWith(letter),
            );
        } else return;
        event.preventDefault();
        next?.focus();
    }

    return (
        <MenuContext.Provider value={{ close }}>
            {trigger({
                ref: triggerRef,
                onClick: () => {
                    setFocusLast(false);
                    setOpen((o) => !o);
                },
                onKeyDown: (event) => {
                    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
                        event.preventDefault();
                        setFocusLast(event.key === "ArrowUp");
                        setOpen(true);
                    }
                },
                "aria-haspopup": "menu",
                "aria-expanded": open,
            })}
            <Popover
                open={open}
                onClose={close}
                anchorRef={triggerRef}
                placement={placement}
            >
                <div ref={listRef} role="menu" onKeyDown={handleListKeyDown}>
                    {children}
                </div>
            </Popover>
        </MenuContext.Provider>
    );
}
