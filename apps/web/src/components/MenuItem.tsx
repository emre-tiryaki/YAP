import type { ReactNode } from "react";
import { useMenu } from "../hooks/useMenu";
import { Kbd } from "./Kbd";

type MenuItemProps = {
    icon?: ReactNode;
    label: string;
    shortcut?: string[];
    danger?: boolean;
    disabled?: boolean;
    onSelect: () => void;
};

export function MenuItem({
    icon,
    label,
    shortcut,
    danger,
    disabled,
    onSelect,
}: MenuItemProps) {
    const { close } = useMenu();
    return (
        <button
            type="button"
            role="menuitem"
            tabIndex={-1}
            disabled={disabled}
            onClick={() => {
                close();
                onSelect();
            }}
            className={`flex h-7 w-full cursor-pointer items-center gap-2 rounded-md px-2 text-left text-body-xs hover:bg-layer-1-hover focus:bg-layer-1-hover focus:outline-none disabled:cursor-default disabled:opacity-50 ${
                danger ? "text-danger" : "text-fg"
            }`}
        >
            {icon && (
                <span className="flex shrink-0 items-center text-fg-subtle">
                    {icon}
                </span>
            )}
            <span className="flex-1 truncate">{label}</span>
            {shortcut && <Kbd keys={shortcut} />}
        </button>
    );
}
