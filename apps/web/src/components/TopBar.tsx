import { Fragment } from "react";
import { Link } from "react-router";
import { TbChevronRight } from "react-icons/tb";
import { useTranslate } from "../hooks/useTranslate";
import type { BreadcrumbItem } from "../types/breadcrumb";

type TopBarProps = {
    path: BreadcrumbItem[];
};

export function TopBar({ path }: TopBarProps) {
    const t = useTranslate();

    return (
        <header className="grid h-14 shrink-0 grid-cols-3 items-center border-b border-gray-200 bg-white px-6">
            <nav aria-label="breadcrumb" className="min-w-0">
                <ol className="flex min-w-0 items-center gap-1 text-sm">
                    {path.map((item, index) => {
                        const isLast = index === path.length - 1;
                        return (
                            <Fragment key={`${index}-${item.href}`}>
                                {index > 0 && (
                                    <li aria-hidden="true" className="shrink-0 text-gray-300">
                                        <TbChevronRight />
                                    </li>
                                )}
                                <li className="min-w-0">
                                    <Link
                                        to={item.href}
                                        aria-current={isLast ? "page" : undefined}
                                        title={item.label}
                                        className={
                                            "block max-w-48 truncate rounded-md px-2 py-1 transition-colors hover:bg-gray-100 hover:text-gray-900 " +
                                            (isLast
                                                ? "font-semibold text-gray-900"
                                                : "text-gray-500")
                                        }
                                    >
                                        {item.label}
                                    </Link>
                                </li>
                            </Fragment>
                        );
                    })}
                </ol>
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
