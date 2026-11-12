import { useId, useState, type KeyboardEvent } from "react";
import { FiSlash } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import { projectIcons } from "../util/projectIcons";

type ProjectIconPickerProps = {
    // A single emoji, or "" for none.
    value: string;
    onChange: (icon: string) => void;
};

const optionClass =
    "grid size-9 cursor-pointer place-items-center rounded-md text-xl hover:bg-gray-100 has-checked:bg-gray-100 has-checked:ring-2 has-checked:ring-gray-900 has-focus-visible:ring-2 has-focus-visible:ring-gray-500";

export function ProjectIconPicker({ value, onChange }: ProjectIconPickerProps) {
    const t = useTranslate();
    const name = useId();
    const isCustom = value !== "" && !projectIcons.includes(value);
    // The custom field is committed on Enter or blur, not on every keystroke,
    // so a half-typed value is never reported.
    const [draft, setDraft] = useState(isCustom ? value : "");

    function commitDraft() {
        const icon = draft.trim();
        if (icon !== "" && icon !== value) onChange(icon);
    }

    function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
        if (event.key !== "Enter") return;
        // Inside a form, Enter would otherwise submit it.
        event.preventDefault();
        commitDraft();
    }

    return (
        <fieldset className="flex flex-col gap-1 text-sm text-gray-700">
            <legend className="mb-1">{t("form.icon")}</legend>
            <div className="flex flex-wrap gap-1">
                <label title={t("form.icon.none")} className={optionClass}>
                    <input
                        type="radio"
                        name={name}
                        checked={value === ""}
                        onChange={() => onChange("")}
                        aria-label={t("form.icon.none")}
                        className="sr-only"
                    />
                    <FiSlash aria-hidden className="text-base text-gray-400" />
                </label>
                {projectIcons.map((icon) => (
                    <label key={icon} className={optionClass}>
                        <input
                            type="radio"
                            name={name}
                            checked={value === icon}
                            onChange={() => onChange(icon)}
                            aria-label={icon}
                            className="sr-only"
                        />
                        <span aria-hidden>{icon}</span>
                    </label>
                ))}
            </div>
            <label className="mt-1 flex items-center gap-2 text-xs text-gray-500">
                {t("form.icon.custom")}
                <input
                    value={draft}
                    onChange={(event) => setDraft(event.target.value)}
                    onBlur={commitDraft}
                    onKeyDown={handleKeyDown}
                    placeholder="🙂"
                    maxLength={16}
                    className={`w-16 rounded-md border bg-white px-2 py-1 text-center text-base ${
                        isCustom ? "border-gray-900" : "border-gray-300"
                    }`}
                />
            </label>
        </fieldset>
    );
}
