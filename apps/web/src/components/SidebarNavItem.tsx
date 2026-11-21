import type { ComponentType, DragEvent } from "react";
import { Link } from "react-router";
import { FiX } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import type { Pin } from "../types/pin";
import { setPinDragData } from "../util/pinDrag";
import { Tooltip } from "./Tooltip";

type IconType = ComponentType<{ className?: string; "aria-hidden"?: boolean }>;

type SidebarNavItemProps = {
    to: string;
    label: string;
    icon: IconType;
    // Shown instead of the icon when set, e.g. a project's emoji.
    emoji?: string;
    active: boolean;
    collapsed: boolean;
    // Nested under a project: indented, no icon.
    nested?: boolean;
    onUnpin?: () => void;
    // Dragging the link carries this pin, so it can be dropped on Pinned.
    dragPin?: Pin;
};

export function SidebarNavItem({
    to,
    label,
    icon: Icon,
    emoji,
    active,
    collapsed,
    nested,
    onUnpin,
    dragPin,
}: SidebarNavItemProps) {
    const t = useTranslate();

    const link = (
        <Link
            to={to}
            onDragStart={
                dragPin
                    ? (event: DragEvent) => setPinDragData(event, dragPin)
                    : undefined
            }
            aria-current={active ? "page" : undefined}
            className={`flex h-7 items-center gap-2 rounded-md text-fg-muted hover:bg-layer-1-hover aria-[current=page]:bg-layer-1-hover aria-[current=page]:font-medium aria-[current=page]:text-fg ${
                collapsed ? "justify-center px-2" : nested ? "pr-2 pl-8" : "px-2"
            } ${onUnpin && !collapsed ? "pr-8" : ""}`}
        >
            {nested ? null : emoji ? (
                <span
                    aria-hidden
                    className="w-4 shrink-0 text-center text-base leading-none"
                >
                    {emoji}
                </span>
            ) : (
                <Icon
                    aria-hidden
                    className={`shrink-0 text-base ${active ? "text-accent" : "text-fg-subtle"}`}
                />
            )}
            {collapsed ? (
                <span className="sr-only">{label}</span>
            ) : (
                <span className="truncate">{label}</span>
            )}
        </Link>
    );

    return (
        <li className="group relative">
            {collapsed ? (
                <Tooltip label={label} side="right">
                    {link}
                </Tooltip>
            ) : (
                link
            )}
            {onUnpin && !collapsed && (
                <button
                    type="button"
                    onClick={onUnpin}
                    aria-label={`${t("card.menu.unpin")}: ${label}`}
                    title={t("card.menu.unpin")}
                    className="absolute top-1/2 right-1.5 hidden -translate-y-1/2 cursor-pointer rounded p-1 text-fg-placeholder group-focus-within:block group-hover:block hover:bg-layer-3-hover hover:text-fg"
                >
                    <FiX aria-hidden />
                </button>
            )}
        </li>
    );
}
