import type { ReactNode } from "react";

type TaskPropertyRowProps = {
    icon: ReactNode;
    label: string;
    children: ReactNode;
};

// One label/value row of the properties column of the full task view.
export function TaskPropertyRow({ icon, label, children }: TaskPropertyRowProps) {
    return (
        <div className="flex min-h-8 items-center gap-2 text-body-xs">
            <span aria-hidden className="flex shrink-0 items-center text-fg-subtle">
                {icon}
            </span>
            <span className="w-28 shrink-0 text-fg-subtle">{label}</span>
            <div className="min-w-0 flex-1">{children}</div>
        </div>
    );
}
