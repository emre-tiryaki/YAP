import type { SyntheticEvent } from "react";
import type { IconType } from "react-icons";
import { FiMoreVertical } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";
import { MenuSeparator } from "./MenuSeparator";

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

    return (
        <div
            onPointerDown={stop}
            onClick={stop}
            onDoubleClick={stop}
            onKeyDown={stop}
            draggable={false}
            onDragStart={(event) => event.preventDefault()}
        >
            <Menu
                placement="bottom-end"
                trigger={({ ref, ...props }) => (
                    <button
                        ref={ref}
                        type="button"
                        aria-label={t("resources.menu.label")}
                        title={t("resources.menu.label")}
                        {...props}
                        className={`cursor-pointer rounded p-1 text-fg-subtle hover:bg-layer-1-hover hover:text-fg focus:opacity-100 focus:outline-2 focus:outline-accent ${
                            isSubtle && !props["aria-expanded"]
                                ? "opacity-0 group-hover:opacity-100"
                                : ""
                        }`}
                    >
                        <FiMoreVertical aria-hidden />
                    </button>
                )}
            >
                {items.map((item) => (
                    <div key={item.key}>
                        {item.startsGroup && <MenuSeparator />}
                        <MenuItem
                            icon={<item.Icon aria-hidden />}
                            label={item.label}
                            danger={item.isDanger}
                            onSelect={item.onSelect}
                        />
                    </div>
                ))}
            </Menu>
        </div>
    );
}
