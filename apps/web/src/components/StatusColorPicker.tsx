import { useTranslate } from "../hooks/useTranslate";
import { statusColors } from "../util/statusColors";

type StatusColorPickerProps = {
    value: string;
    onChange: (color: string) => void;
};

export function StatusColorPicker({ value, onChange }: StatusColorPickerProps) {
    const t = useTranslate();
    const isCustom = !statusColors.includes(value.toLowerCase());

    return (
        <fieldset className="flex flex-col gap-1 text-sm text-gray-700">
            <legend className="mb-1">{t("plan.status.create.color")}</legend>
            <div className="flex flex-wrap items-center gap-2">
                {statusColors.map((c) => (
                    <label
                        key={c}
                        className="cursor-pointer rounded-full has-checked:ring-2 has-checked:ring-gray-900 has-checked:ring-offset-2 has-focus-visible:ring-2 has-focus-visible:ring-gray-500"
                    >
                        <input
                            type="radio"
                            name="status-color"
                            value={c}
                            checked={value.toLowerCase() === c}
                            onChange={() => onChange(c)}
                            aria-label={c}
                            className="sr-only"
                        />
                        <span
                            aria-hidden
                            className="block h-6 w-6 rounded-full"
                            style={{ backgroundColor: c }}
                        />
                    </label>
                ))}
                <label
                    title={t("plan.status.color.custom")}
                    className={`relative block h-6 w-6 cursor-pointer overflow-hidden rounded-full border border-gray-300 has-focus-visible:ring-2 has-focus-visible:ring-gray-500 ${
                        isCustom ? "ring-2 ring-gray-900 ring-offset-2" : ""
                    }`}
                    style={{
                        background: isCustom
                            ? value
                            : "conic-gradient(#ef4444, #eab308, #22c55e, #3b82f6, #a855f7, #ef4444)",
                    }}
                >
                    <input
                        type="color"
                        value={value}
                        onChange={(event) => onChange(event.target.value)}
                        aria-label={t("plan.status.color.custom")}
                        className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
                    />
                </label>
            </div>
        </fieldset>
    );
}
