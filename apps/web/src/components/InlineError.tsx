import { useTranslate } from "../hooks/useTranslate";

type InlineErrorProps = {
    message: string;
    onRetry?: () => void;
};

export function InlineError({ message, onRetry }: InlineErrorProps) {
    const t = useTranslate();

    return (
        <div
            role="alert"
            className="flex items-center justify-between gap-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
        >
            <span>{message}</span>
            {onRetry && (
                <button
                    type="button"
                    onClick={onRetry}
                    className="cursor-pointer rounded-md border border-red-300 bg-white px-3 py-1.5 font-medium text-red-700 hover:bg-red-100"
                >
                    {t("common.retry")}
                </button>
            )}
        </div>
    );
}
