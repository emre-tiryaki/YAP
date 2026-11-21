import { Fragment, type ReactNode } from "react";
import { Link } from "react-router";
import { FiChevronRight } from "react-icons/fi";
import type { BreadcrumbItem } from "../types/breadcrumb";
import { FiSearch } from "react-icons/fi";
import { openPalette } from "../hooks/useCommandPalette";
import { useTranslate } from "../hooks/useTranslate";
import { Button } from "./Button";
import { Kbd } from "./Kbd";

type PageHeaderProps = {
    breadcrumb: BreadcrumbItem[];
    actions?: ReactNode;
    children?: ReactNode;
};

// The bar on top of every page: breadcrumb on the left, search and the page's
// actions on the right. It is a container, so HeaderAction labels collapse to
// icons when the page is narrow.
export function PageHeader({ breadcrumb, actions, children }: PageHeaderProps) {
    const t = useTranslate();
    return (
        <header className="@container flex h-12 shrink-0 items-center gap-2 border-b border-line px-4">
            <nav aria-label="Breadcrumb" className="min-w-0">
                <ol className="flex min-w-0 items-center gap-1 text-body-xs">
                    {breadcrumb.map((item, index) => {
                        const isLast = index === breadcrumb.length - 1;
                        const content = (
                            <>
                                {item.icon && (
                                    <span className="flex shrink-0 items-center">
                                        {item.icon}
                                    </span>
                                )}
                                <span className="truncate">{item.label}</span>
                            </>
                        );
                        const className = `flex max-w-48 items-center gap-1.5 rounded-md px-1.5 py-1 ${
                            isLast
                                ? "font-medium text-fg"
                                : "text-fg-muted hover:text-fg"
                        }`;
                        return (
                            <Fragment key={`${index}-${item.to ?? item.label}`}>
                                {index > 0 && (
                                    <li
                                        aria-hidden
                                        className="shrink-0 text-fg-placeholder"
                                    >
                                        <FiChevronRight />
                                    </li>
                                )}
                                <li className="min-w-0">
                                    {item.to ? (
                                        <Link
                                            to={item.to}
                                            aria-current={isLast ? "page" : undefined}
                                            title={item.label}
                                            className={className}
                                        >
                                            {content}
                                        </Link>
                                    ) : (
                                        <span
                                            aria-current={isLast ? "page" : undefined}
                                            className={className}
                                        >
                                            {content}
                                        </span>
                                    )}
                                </li>
                            </Fragment>
                        );
                    })}
                </ol>
            </nav>
            <div className="flex-1" />
            <Button
                variant="secondary"
                size="sm"
                leadingIcon={FiSearch}
                aria-label={t("palette.search")}
                onClick={() => openPalette()}
            >
                <span className="hidden @3xl:inline">{t("palette.search")}</span>
                <span className="hidden @3xl:inline">
                    <Kbd keys={["mod", "K"]} />
                </span>
            </Button>
            {children}
            {actions}
        </header>
    );
}
