import { useState, type DragEvent, type ReactNode } from "react";
import { Link } from "react-router";
import { FiChevronDown, FiChevronRight } from "react-icons/fi";

type SidebarSectionProps = {
    title: string;
    // Makes the title a link, e.g. to the page listing the section's items.
    to?: string;
    collapsed: boolean;
    // Makes the section a drop target; dropState styles it while a matching
    // item is dragged ("active") or held over it ("over").
    dropProps?: {
        onDragOver: (event: DragEvent) => void;
        onDragLeave: (event: DragEvent) => void;
        onDrop: (event: DragEvent) => void;
    };
    dropState?: "active" | "over";
    children: ReactNode;
};

export function SidebarSection({
    title,
    to,
    collapsed,
    dropProps,
    dropState,
    children,
}: SidebarSectionProps) {
    const [open, setOpen] = useState(true);

    return (
        <section
            aria-label={title}
            {...dropProps}
            className={
                dropProps
                    ? `-m-1 rounded-lg border border-dashed p-1 transition-colors ${
                          dropState === "over"
                              ? "border-accent bg-accent/10"
                              : dropState === "active"
                                ? "border-line-strong bg-layer-1"
                                : "border-transparent"
                      }`
                    : undefined
            }
        >
            {collapsed ? (
                <hr className="mx-2 mb-2 border-line" />
            ) : (
                <h2 className="mb-0.5 flex items-center gap-1 px-2 text-caption font-semibold text-fg-subtle">
                    <button
                        type="button"
                        onClick={() => setOpen(!open)}
                        aria-expanded={open}
                        aria-label={title}
                        className="flex size-4 shrink-0 cursor-pointer items-center justify-center rounded hover:text-fg"
                    >
                        {open ? (
                            <FiChevronDown aria-hidden />
                        ) : (
                            <FiChevronRight aria-hidden />
                        )}
                    </button>
                    {to ? (
                        <Link to={to} className="rounded hover:text-fg">
                            {title}
                        </Link>
                    ) : (
                        title
                    )}
                </h2>
            )}
            {(open || collapsed) && (
                <ul className="flex flex-col gap-0.5">{children}</ul>
            )}
        </section>
    );
}
