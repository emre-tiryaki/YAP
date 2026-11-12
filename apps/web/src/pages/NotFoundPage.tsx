import { Link } from "react-router";
import { ErrorState } from "../components/ErrorState";
import { useTranslate } from "../hooks/useTranslate";

export default function NotFoundPage() {
    const t = useTranslate();

    return (
        <ErrorState
            code="404"
            title={t("error.notFound.title")}
            message={t("error.notFound.message")}
            actions={
                <Link
                    to="/home"
                    className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:bg-accent/85"
                >
                    {t("error.goHome")}
                </Link>
            }
        />
    );
}
