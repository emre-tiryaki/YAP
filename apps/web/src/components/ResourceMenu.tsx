import { useEffect, useRef, useState, type SyntheticEvent } from "react";
import { createPortal } from "react-dom";
import type { IconType } from "react-icons";
import { FiMoreVertical } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";

export type ResourceMenuItem = {
    key: string;
    label: string;
    Icon: IconType;
    onSelect: () => void;
    isDanger?: boolean;
    // A separator is drawn above the item.
    startsGroup?: boolean;
};

type ResourceMenuProps = {
    items: ResourceMenuItem[];
    // Shown on hover only, unless the menu is open or focused.
    isSubtle?: boolean;
};

// Keeps clicks in the menu from selecting or opening the row behind it.
function stop(event: SyntheticEvent) {
    event.stopPropagation();
}

// The ⋮ button next to a resource with its file operations.
export function ResourceMenu({ items, isSubtle = true }: ResourceMenuProps) {
    const t = useTranslate();
    const [anchor, setAnchor] = useState<{ top: number; left: number } | null>(
        null,
    );
    const isOpen = anchor !== null;
    const buttonRef = useRef<HTMLButtonElement>(null);
    const menuRef = useRef<HTMLUListElement>(null);

    useEffect(() => {
        if (!isOpen) return;
        const close = () => setAnchor(null);
        function handlePointerDown(event: PointerEvent) {
            const target = event.target as Node;
            if (
                !buttonRef.current?.contains(target) &&
                !menuRef.current?.contains(target)
            )
                close();
        }
        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") {
                event.stopPropagation();
                close();
                buttonRef.current?.focus();
            }
        }
        document.addEventListener("pointerdown", handlePointerDown, true);
        document.addEventListener("keydown", handleKeyDown, true);
        window.addEventListener("scroll", close, true);
        window.addEventListener("resize", close);
        menuRef.current?.querySelector("button")?.focus();
        return () => {
            document.removeEventListener("pointerdown", handlePointerDown, true);
            document.removeEventListener("keydown", handleKeyDown, true);
            window.removeEventListener("scroll", close, true);
            window.removeEventListener("resize", close);
        };
    }, [isOpen]);

    function toggle() {
        if (isOpen) {
            setAnchor(null);
            return;
        }
        const rect = buttonRef.current!.getBoundingClientRect();
        const width = 208;
        const left = Math.min(rect.right - width, window.innerWidth - width - 8);
        // Opens upwards when there is no room below.
        const height = items.length * 36 + 8;
        const top =
            rect.bottom + height > window.innerHeight
                ? Math.max(8, rect.top - height - 4)
                : rect.bottom + 4;
        setAnchor({ top, left: Math.max(8, left) });
    }

    return (
        <div
            onPointerDown={stop}
            onClick={stop}
            onDoubleClick={stop}
            onKeyDown={stop}
            draggable={false}
            onDragStart={(event) => event.preventDefault()}
        >
            <button
                ref={buttonRef}
                type="button"
                aria-label={t("resources.menu.label")}
                title={t("resources.menu.label")}
                aria-haspopup="menu"
                aria-expanded={isOpen}
                onClick={toggle}
                className={`cursor-pointer rounded p-1 text-gray-500 hover:bg-gray-200 hover:text-gray-900 focus:opacity-100 focus:outline-2 focus:outline-accent ${
                    isSubtle && !isOpen ? "opacity-0 group-hover:opacity-100" : ""
                }`}
            >
                <FiMoreVertical aria-hidden />
            </button>
            {anchor &&
                createPortal(
                    <ul
                        ref={menuRef}
                        role="menu"
                        style={{ top: anchor.top, left: anchor.left }}
                        className="fixed z-50 w-52 rounded-md border border-gray-200 bg-white py-1 text-sm shadow-lg"
                        onKeyDown={(event) => {
                            if (event.key !== "ArrowDown" && event.key !== "ArrowUp")
                                return;
                            event.preventDefault();
                            const buttons = [
                                ...menuRef.current!.querySelectorAll("button"),
                            ];
                            const index = buttons.indexOf(
                                document.activeElement as HTMLButtonElement,
                            );
                            const step = event.key === "ArrowDown" ? 1 : -1;
                            buttons[
                                (index + step + buttons.length) % buttons.length
                            ]?.focus();
                        }}
                    >
                        {items.map((item) => (
                            <li
                                key={item.key}
                                role="none"
                                className={
                                    item.startsGroup
                                        ? "mt-1 border-t border-gray-100 pt-1"
                                        : ""
                                }
                            >
                                <button
                                    type="button"
                                    role="menuitem"
                                    onClick={() => {
                                        setAnchor(null);
                                        item.onSelect();
                                    }}
                                    className={`flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left focus:outline-none ${
                                        item.isDanger
                                            ? "text-red-600 hover:bg-red-50 focus:bg-red-50"
                                            : "text-gray-700 hover:bg-gray-100 focus:bg-gray-100"
                                    }`}
                                >
                                    <item.Icon aria-hidden className="shrink-0" />
                                    {item.label}
                                </button>
                            </li>
                        ))}
                    </ul>,
                    document.body,
                )}
        </div>
    );
}
