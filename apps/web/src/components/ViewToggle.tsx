import type { ComponentType } from "react";
import { FiColumns, FiList } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";

export type BoardView = "board" | "list";

type ViewOption<T extends string> = {
    view: T;
    label: string;
    Icon: ComponentType<{ "aria-hidden"?: boolean }>;
};

type ViewToggleProps<T extends string> = {
    value: T;
    onChange: (view: T) => void;
    // Defaults to the plan page's board/list options.
    options?: readonly ViewOption<T>[];
};

export function ViewToggle<T extends string = BoardView>({
    value,
    onChange,
    options,
}: ViewToggleProps<T>) {
    const t = useTranslate();
    const boardOptions = [
        { view: "board", label: t("plan.view.board"), Icon: FiColumns },
        { view: "list", label: t("plan.view.list"), Icon: FiList },
    ] as unknown as readonly ViewOption<T>[];

    return (
        <div
            role="radiogroup"
            aria-label={t("plan.view.label")}
            className="inline-flex rounded-md border border-gray-300 bg-white p-0.5"
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
                            : "text-gray-600 hover:bg-gray-100"
                    }`}
                >
                    <Icon aria-hidden />
                    {label}
                </button>
            ))}
        </div>
    );
}
