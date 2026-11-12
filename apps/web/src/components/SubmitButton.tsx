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
            className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:bg-accent/85 disabled:opacity-50"
        >
            {children}
        </button>
    )
}