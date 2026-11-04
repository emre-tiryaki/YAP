import type { ReactNode } from "react"

type SubmitButtonProps = {
    children: ReactNode
    disabled?: boolean
}

export function SubmitButton({ children, disabled }: SubmitButtonProps) {
    return (
        <button
            type="submit"
            disabled={disabled}
            className="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:opacity-50"
        >
            {children}
        </button>
    )
}