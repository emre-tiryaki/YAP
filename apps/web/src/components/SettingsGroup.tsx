import type { ReactNode } from "react";

export function SettingsGroup({
    title,
    children,
}: {
    title?: string;
    children: ReactNode;
}) {
    return (
        <section aria-label={title}>
            {title && (
                <h4 className="mb-2 text-xs font-semibold tracking-wide text-fg-subtle uppercase">
                    {title}
                </h4>
            )}
            <div className="divide-y divide-line rounded-lg border border-line bg-layer-2">
                {children}
            </div>
        </section>
    );
}
