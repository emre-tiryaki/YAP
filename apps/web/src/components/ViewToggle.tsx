import type { ComponentType } from "react";
import { FiCalendar, FiColumns, FiGrid, FiList } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import { Tooltip } from "./Tooltip";

export type BoardView = "board" | "list" | "calendar" | "table";

type ViewOption<T extends string> = {
    view: T;
    label: string;
    // Without an icon the option is text only.
    Icon?: ComponentType<{ "aria-hidden"?: boolean }>;
};

type ViewToggleProps<T extends string> = {
    value: T;
    onChange: (view: T) => void;
    // Defaults to the plan page's board/list options.
    options?: readonly ViewOption<T>[];
    // Accessible name of the group; defaults to the plan view label.
    label?: string;
    // Shows only the icons; the labels stay as accessible names and tooltips.
    iconOnly?: boolean;
};

export function ViewToggle<T extends string = BoardView>({
    value,
    onChange,
    options,
    label,
    iconOnly = false,
}: ViewToggleProps<T>) {
    const t = useTranslate();
    const boardOptions = [
        { view: "board", label: t("plan.view.board"), Icon: FiColumns },
        { view: "list", label: t("plan.view.list"), Icon: FiList },
        { view: "calendar", label: t("plan.view.calendar"), Icon: FiCalendar },
        { view: "table", label: t("plan.view.table"), Icon: FiGrid },
    ] as unknown as readonly ViewOption<T>[];

    return (
        <div
            role="radiogroup"
            aria-label={label ?? t("plan.view.label")}
            className="inline-flex rounded-md border border-line-strong bg-layer-2 p-0.5"
        >
            {(options ?? boardOptions).map(({ view, label, Icon }) => {
                const button = (
                    <button
                        key={view}
                        type="button"
                        role="radio"
                        aria-checked={value === view}
                        aria-label={iconOnly ? label : undefined}
                        onClick={() => onChange(view)}
                        className={`flex cursor-pointer items-center gap-1.5 rounded text-sm ${
                            iconOnly ? "px-2 py-1" : "px-3 py-1"
                        } ${
                            value === view
                                ? "bg-accent text-accent-fg"
                                : "text-fg-muted hover:bg-layer-1-hover"
                        }`}
                    >
                        {Icon && <Icon aria-hidden />}
                        {!iconOnly && label}
                    </button>
                );
                return iconOnly ? (
                    <Tooltip key={view} label={label}>
                        {button}
                    </Tooltip>
                ) : (
                    button
                );
            })}
        </div>
    );
}
