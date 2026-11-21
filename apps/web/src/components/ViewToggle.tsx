import type { ComponentType } from "react";
import { FiColumns, FiList } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";

export type BoardView = "board" | "list";

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
};

export function ViewToggle<T extends string = BoardView>({
    value,
    onChange,
    options,
    label,
}: ViewToggleProps<T>) {
    const t = useTranslate();
    const boardOptions = [
        { view: "board", label: t("plan.view.board"), Icon: FiColumns },
        { view: "list", label: t("plan.view.list"), Icon: FiList },
    ] as unknown as readonly ViewOption<T>[];

    return (
        <div
            role="radiogroup"
            aria-label={label ?? t("plan.view.label")}
            className="inline-flex rounded-md border border-line-strong bg-layer-2 p-0.5"
        >
            {(options ?? boardOptions).map(({ view, label, Icon }) => (
                <button
                    key={view}
                    type="button"
                    role="radio"
                    aria-checked={value === view}
                    onClick={() => onChange(view)}
                    className={`flex cursor-pointer items-center gap-1.5 rounded px-3 py-1 text-sm ${
                        value === view
                            ? "bg-accent text-accent-fg"
                            : "text-fg-muted hover:bg-layer-1-hover"
                    }`}
                >
                    {Icon && <Icon aria-hidden />}
                    {label}
                </button>
            ))}
        </div>
    );
}
