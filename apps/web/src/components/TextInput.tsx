import type { InputHTMLAttributes } from "react"

type TextInputProps = {
    label: string
} & InputHTMLAttributes<HTMLInputElement>

export function TextInput({ label, ...inputProps }: TextInputProps) {
    return (
        <label className="flex flex-col gap-1 text-sm text-gray-700">
            {label}
            <input className="rounded-md border border-gray-300 bg-white px-3 py-2" {...inputProps} />
        </label>
    )
}