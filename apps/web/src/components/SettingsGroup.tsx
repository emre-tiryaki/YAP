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
                <h4 className="mb-2 text-xs font-semibold tracking-wide text-gray-400 uppercase">
                    {title}
                </h4>
            )}
            <div className="divide-y divide-gray-100 rounded-lg border border-gray-200">
                {children}
            </div>
        </section>
    );
}
