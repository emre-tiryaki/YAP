import { isRouteErrorResponse, Link, useRouteError } from "react-router";
import { ErrorState } from "../components/ErrorState";
import { useTranslate } from "../hooks/useTranslate";

export default function RouteErrorPage() {
    const t = useTranslate();
    const error = useRouteError();

    const isNotFound = isRouteErrorResponse(error) && error.status === 404;
    const code = isRouteErrorResponse(error) ? String(error.status) : undefined;

    return (
        <ErrorState
            code={code}
            title={
                isNotFound
                    ? t("error.notFound.title")
                    : t("error.unexpected.title")
            }
            message={
                isNotFound
                    ? t("error.notFound.message")
                    : t("error.unexpected.message")
            }
            actions={
                <>
                    <button
                        type="button"
                        onClick={() => window.location.reload()}
                        className="cursor-pointer rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-900 hover:bg-gray-100"
                    >
                        {t("error.retry")}
                    </button>
                    <Link
                        to="/home"
                        className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:bg-accent/85"
                    >
                        {t("error.goHome")}
                    </Link>
                </>
            }
        />
    );
}
