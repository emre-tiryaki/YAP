import { useState, type ReactNode } from "react";
import { FiChevronDown, FiChevronRight } from "react-icons/fi";

type TaskSectionProps = {
    title: string;
    count?: number;
    children: ReactNode;
};

// A collapsible block of the task body (subtasks, resources, comments).
export function TaskSection({ title, count, children }: TaskSectionProps) {
    const [open, setOpen] = useState(true);

    return (
        <section className="border-t border-line pt-1">
            <h3>
                <button
                    type="button"
                    aria-expanded={open}
                    onClick={() => setOpen(!open)}
                    className="flex h-8 w-full cursor-pointer items-center gap-1.5 text-body-xs font-medium text-fg hover:text-fg-muted"
                >
                    {open ? (
                        <FiChevronDown aria-hidden />
                    ) : (
                        <FiChevronRight aria-hidden />
                    )}
                    {title}
                    {count !== undefined && count > 0 && (
                        <span className="font-normal text-fg-subtle">{count}</span>
                    )}
                </button>
            </h3>
            {open && <div className="pt-2 pb-3">{children}</div>}
        </section>
    );
}
