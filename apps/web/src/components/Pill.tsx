import type { ReactNode } from "react";

type PillProps = {
    icon?: ReactNode;
    children: ReactNode;
    // Tailwind text color class for the icon, e.g. "text-red-600".
    color?: string;
    className?: string;
};

// Static, read-only twin of the PropertySelect pill.
export function Pill({ icon, children, color = "", className = "" }: PillProps) {
    return (
        <span
            className={`inline-flex h-6 items-center gap-1 rounded-md border border-line bg-layer-2 px-1.5 text-caption text-fg-muted ${className}`}
        >
            {icon && (
                <span className={`flex shrink-0 items-center ${color}`}>
                    {icon}
                </span>
            )}
            {children}
        </span>
    );
}
