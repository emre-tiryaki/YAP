import { useTranslate } from "../hooks/useTranslate";

type TopBarProps = {
    path: string[];
};

export function TopBar({ path }: TopBarProps) {
    const t = useTranslate();

    return (
        <header className="grid h-14 shrink-0 grid-cols-3 items-center border-b border-gray-200 bg-white px-6">
            <nav
                aria-label="breadcrumb"
                className="truncate text-sm text-gray-600"
            >
                {path.map((segment) => `/${segment}`).join("")}
            </nav>
            <input
                type="search"
                placeholder={t("topbar.search")}
                aria-label={t("topbar.search")}
                className="w-full rounded-md border border-gray-300 bg-gray-50 px-3 py-1.5 text-sm"
            />
            <div />
        </header>
    );
}
