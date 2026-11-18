import { Link } from "react-router";
import { useTranslate } from "../hooks/useTranslate";

type ViewAllLinkProps = {
    href: string;
    // The full number of items, of which only a part is shown.
    total: number;
};

// Shown under a shortened home list, leading to the page with all items.
export function ViewAllLink({ href, total }: ViewAllLinkProps) {
    const t = useTranslate();

    return (
        <Link
            to={href}
            className="mt-3 inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm font-medium text-accent hover:bg-accent/10 focus-visible:outline-2 focus-visible:outline-accent"
        >
            {t("common.viewAll")}
            <span className="text-gray-400 tabular-nums">({total})</span>
        </Link>
    );
}
