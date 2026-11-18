import { Link } from "react-router";
import { FiChevronRight } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";

type SectionHeaderProps = {
    title: string;
    onCreate: () => void;
    // Makes the title a link, e.g. to the section's own page.
    href?: string;
};

export function SectionHeader({ title, onCreate, href }: SectionHeaderProps) {
    const t = useTranslate();

    return (
        <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold">
                {href ? (
                    <Link
                        to={href}
                        className="group inline-flex items-center gap-1 rounded-md hover:text-accent focus-visible:outline-2 focus-visible:outline-accent"
                    >
                        {title}
                        <FiChevronRight
                            aria-hidden
                            className="text-gray-400 transition-transform group-hover:translate-x-0.5 group-hover:text-accent"
                        />
                    </Link>
                ) : (
                    title
                )}
            </h2>
            <button
                type="button"
                onClick={onCreate}
                className="cursor-pointer rounded-md bg-accent px-3 py-1.5 text-sm font-medium text-accent-fg hover:bg-accent/85"
            >
                + {t("common.new")}
            </button>
        </div>
    );
}
