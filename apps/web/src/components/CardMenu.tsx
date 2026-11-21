import type { SyntheticEvent } from "react";
import { FiEdit2, FiMoreVertical, FiTrash2, FiType } from "react-icons/fi";
import { TbPin, TbPinnedOff } from "react-icons/tb";
import { togglePin, usePins } from "../hooks/usePins";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Pin } from "../types/pin";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";

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

    return (
        <div
            className="relative"
            onPointerDown={stop}
            onKeyDown={stop}
            onClick={stop}
        >
            <Menu
                placement="bottom-end"
                trigger={({ ref, ...props }) => (
                    <button
                        ref={ref}
                        type="button"
                        aria-label={t("card.menu.label")}
                        title={t("card.menu.label")}
                        {...props}
                        className="cursor-pointer rounded p-1 text-fg-placeholder hover:bg-layer-1-hover hover:text-fg focus:outline-2 focus:outline-accent"
                    >
                        <FiMoreVertical aria-hidden />
                    </button>
                )}
            >
                {pin && (
                    <MenuItem
                        icon={pinned ? <TbPinnedOff aria-hidden /> : <TbPin aria-hidden />}
                        label={t(pinned ? "card.menu.unpin" : "card.menu.pin")}
                        onSelect={() => togglePin(pin)}
                    />
                )}
                {items.map(({ action, label, Icon }) => (
                    <MenuItem
                        key={action}
                        icon={<Icon aria-hidden />}
                        label={t(label)}
                        danger={action === "delete"}
                        onSelect={() => onSelect(action)}
                    />
                ))}
            </Menu>
        </div>
    );
}
