import { FiColumns, FiList } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";

export type BoardView = "board" | "list";

type ViewToggleProps = {
    value: BoardView;
    onChange: (view: BoardView) => void;
};

export function ViewToggle({ value, onChange }: ViewToggleProps) {
    const t = useTranslate();
    const options = [
        { view: "board", label: t("plan.view.board"), Icon: FiColumns },
        { view: "list", label: t("plan.view.list"), Icon: FiList },
    ] as const;

    return (
        <div
            role="radiogroup"
            aria-label={t("plan.view.label")}
            className="inline-flex rounded-md border border-gray-300 bg-white p-0.5"
        >
            {options.map(({ view, label, Icon }) => (
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
