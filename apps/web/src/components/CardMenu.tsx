import { useEffect, useRef, useState, type SyntheticEvent } from "react";
import { createPortal } from "react-dom";
import { FiEdit2, FiMoreVertical, FiTrash2, FiType } from "react-icons/fi";
import { TbPin, TbPinnedOff } from "react-icons/tb";
import { togglePin, usePins } from "../hooks/usePins";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Pin } from "../types/pin";

type CardMenuProps = {
    onSelect: (action: CardAction) => void;
    // When given, the menu also offers pinning the card to the sidebar.
    pin?: Pin;
};

const items = [
    { action: "rename", label: "card.menu.rename", Icon: FiType },
    { action: "edit", label: "card.menu.edit", Icon: FiEdit2 },
    { action: "delete", label: "card.menu.delete", Icon: FiTrash2 },
] as const;

// Keeps events inside the menu from reaching the card: no link navigation and
// no drag start on sortable cards.
function stop(event: SyntheticEvent) {
    event.stopPropagation();
}

export function CardMenu({ onSelect, pin }: CardMenuProps) {
    const t = useTranslate();
    const { isPinned } = usePins();
    const pinned = pin !== undefined && isPinned(pin.kind, pin.id);
    // Viewport position of the open menu; null while closed. The menu is
    // portalled to <body> so scrolling card lists cannot clip it.
    const [anchor, setAnchor] = useState<{ top: number; right: number } | null>(
        null,
    );
    const isOpen = anchor !== null;
    const ref = useRef<HTMLDivElement>(null);
    const menuRef = useRef<HTMLUListElement>(null);

    useEffect(() => {
        if (!isOpen) return;

        const close = () => setAnchor(null);
        function handlePointerDown(event: PointerEvent) {
            const target = event.target as Node;
            if (
                !ref.current?.contains(target) &&
                !menuRef.current?.contains(target)
            )
                close();
        }
        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") close();
        }

        // Capture phase, so menus close even when the click lands on an
        // element that stops propagation (e.g. another card's menu).
        document.addEventListener("pointerdown", handlePointerDown, true);
        document.addEventListener("keydown", handleKeyDown, true);
        // A fixed menu would drift away from its button.
        window.addEventListener("scroll", close, true);
        window.addEventListener("resize", close);
        return () => {
            document.removeEventListener(
                "pointerdown",
                handlePointerDown,
                true,
            );
            document.removeEventListener("keydown", handleKeyDown, true);
            window.removeEventListener("scroll", close, true);
            window.removeEventListener("resize", close);
        };
    }, [isOpen]);

    return (
        <div
            ref={ref}
            className="relative"
            onPointerDown={stop}
            onKeyDown={stop}
            onClick={stop}
        >
            <button
                type="button"
                aria-label={t("card.menu.label")}
                title={t("card.menu.label")}
                aria-haspopup="menu"
                aria-expanded={isOpen}
                onClick={(event) => {
                    if (isOpen) {
                        setAnchor(null);
                        return;
                    }
                    const rect = event.currentTarget.getBoundingClientRect();
                    setAnchor({
                        top: rect.bottom + 4,
                        right: window.innerWidth - rect.right,
                    });
                }}
                className="cursor-pointer rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-900 focus:outline-2 focus:outline-gray-900"
            >
                <FiMoreVertical aria-hidden />
            </button>
            {anchor &&
                createPortal(
                    <ul
                        ref={menuRef}
                        role="menu"
                        style={{ top: anchor.top, right: anchor.right }}
                        className="fixed z-40 w-44 rounded-md border border-gray-200 bg-white py-1 text-sm shadow-lg"
                    >
                        {pin && (
                            <li role="none">
                                <button
                                    type="button"
                                    role="menuitem"
                                    onClick={() => {
                                        setAnchor(null);
                                        togglePin(pin);
                                    }}
                                    className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-gray-700 hover:bg-gray-100"
                                >
                                    {pinned ? (
                                        <TbPinnedOff aria-hidden />
                                    ) : (
                                        <TbPin aria-hidden />
                                    )}
                                    {t(pinned ? "card.menu.unpin" : "card.menu.pin")}
                                </button>
                            </li>
                        )}
                        {items.map(({ action, label, Icon }) => (
                            <li key={action} role="none">
                                <button
                                    type="button"
                                    role="menuitem"
                                    onClick={() => {
                                        setAnchor(null);
                                        onSelect(action);
                                    }}
                                    className={`flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left ${
                                        action === "delete"
                                            ? "text-red-600 hover:bg-red-50"
                                            : "text-gray-700 hover:bg-gray-100"
                                    }`}
                                >
                                    <Icon aria-hidden />
                                    {t(label)}
                                </button>
                            </li>
                        ))}
                    </ul>,
                    document.body,
                )}
        </div>
    );
}
