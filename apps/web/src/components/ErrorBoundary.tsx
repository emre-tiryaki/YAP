import { Component, type ErrorInfo, type ReactNode } from "react";
import { Link } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import { Button } from "./Button";
import { ErrorState } from "./ErrorState";

function ErrorFallback() {
    const t = useTranslate();
    return (
        <div className="min-h-0 flex-1 overflow-auto">
            <ErrorState
                title={t("error.unexpected.title")}
                message={t("error.unexpected.message")}
                actions={
                    <>
                        <Button
                            size="lg"
                            onClick={() => window.location.reload()}
                        >
                            {t("error.retry")}
                        </Button>
                        <Link
                            to="/home"
                            className="inline-flex h-8 items-center rounded-md bg-accent px-3 text-sm font-medium text-accent-fg hover:opacity-90"
                        >
                            {t("error.goHome")}
                        </Link>
                    </>
                }
            />
        </div>
    );
}

type ErrorBoundaryProps = { children: ReactNode };

// Catches render errors of the page below it so the shell (sidebar) stays
// usable. Mount it with a `key` that changes on navigation to reset it.
export class ErrorBoundary extends Component<
    ErrorBoundaryProps,
    { failed: boolean }
> {
    state = { failed: false };

    static getDerivedStateFromError() {
        return { failed: true };
    }

    componentDidCatch(error: Error, info: ErrorInfo) {
        console.error(error, info.componentStack);
    }

    render() {
        return this.state.failed ? <ErrorFallback /> : this.props.children;
    }
}
