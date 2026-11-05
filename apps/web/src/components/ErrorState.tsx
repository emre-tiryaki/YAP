import type { ReactNode } from "react";

type ErrorStateProps = {
    code?: string;
    title: string;
    message: string;
    actions?: ReactNode;
};

export function ErrorState({ code, title, message, actions }: ErrorStateProps) {
    return (
        <main
            role="alert"
            className="flex min-h-screen flex-col items-center justify-center gap-4 bg-gray-50 p-8 text-center"
        >
            {code && (
                <p className="text-6xl font-bold tracking-tight text-gray-900">
                    {code}
                </p>
            )}
            <h1 className="text-2xl font-semibold">{title}</h1>
            <p className="max-w-md text-sm text-gray-600">{message}</p>
            {actions && <div className="mt-2 flex gap-3">{actions}</div>}
        </main>
    );
}
