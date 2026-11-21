import type { ComponentType, ReactNode } from "react";

type EmptyStateProps = {
    icon: ComponentType<{ "aria-hidden"?: boolean; className?: string }>;
    title: string;
    description?: string;
    action?: ReactNode;
    compact?: boolean;
};

export function EmptyState({
    icon: Icon,
    title,
    description,
    action,
    compact,
}: EmptyStateProps) {
    if (compact) {
        return (
            <div className="flex items-center justify-center gap-2 py-6 text-body-xs text-fg-subtle">
                <Icon aria-hidden className="size-5 text-fg-placeholder" />
                <span>{title}</span>
                {action}
            </div>
        );
    }
    return (
        <div className="mx-auto flex max-w-100 flex-col items-start gap-2 py-16">
            <Icon aria-hidden className="size-10 text-fg-placeholder" />
            <h2 className="text-base font-semibold text-fg">{title}</h2>
            {description && (
                <p className="text-body-xs text-fg-subtle">{description}</p>
            )}
            {action && <div className="mt-2">{action}</div>}
        </div>
    );
}
