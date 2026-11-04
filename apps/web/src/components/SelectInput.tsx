import type { SelectHTMLAttributes } from "react";

type SelectOption = {
    value: string;
    label: string;
};

type SelectInputProps = {
    label: string;
    options: SelectOption[];
} & SelectHTMLAttributes<HTMLSelectElement>;

export function SelectInput({
    label,
    options,
    ...selectProps
}: SelectInputProps) {
    return (
        <label className="flex flex-col gap-1 text-sm text-gray-700">
            {label}
            <select
                className="rounded-md border border-gray-300 bg-white px-3 py-2"
                {...selectProps}
            >
                {options.map((option) => (
                    <option key={option.value} value={option.value}>
                        {option.label}
                    </option>
                ))}
            </select>
        </label>
    );
}
