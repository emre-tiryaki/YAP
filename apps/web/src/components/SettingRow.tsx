import type { ReactNode } from "react";

export function SettingRow({
    label,
    hint,
    hintTone = "default",
    children,
}: {
    label: string;
    hint?: string;
    hintTone?: "default" | "error";
    children: ReactNode;
}) {
    return (
        <div className="flex items-center justify-between gap-6 px-5 py-4">
            <div className="min-w-0">
                <p className="text-sm font-medium text-gray-900">{label}</p>
                {hint && (
                    <p
                        role={hintTone === "error" ? "alert" : undefined}
                        className={`mt-0.5 text-xs ${
                            hintTone === "error" ? "text-red-600" : "text-gray-500"
                        }`}
                    >
                        {hint}
                    </p>
                )}
            </div>
            <div className="shrink-0">{children}</div>
        </div>
    );
}
