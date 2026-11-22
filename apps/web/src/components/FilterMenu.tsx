import { useRef, useState } from "react";
import { FiCheck, FiFilter } from "react-icons/fi";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { Label } from "../types/label";
import type { Status } from "../types/status";
import type { TaskPriority } from "../types/task";
import type { DoneFilter, DueFilter, TaskFilters } from "../types/taskView";
import { labelDotClass } from "../util/labelColors";
import { priorities } from "../util/priority";
import { hasActiveFilters } from "../util/taskFilter";
import { HeaderAction } from "./HeaderAction";
import { Popover } from "./Popover";

type FilterMenuProps = {
    filters: TaskFilters;
    onChange: (filters: TaskFilters) => void;
    // Left out (no section) when not given or empty.
    statuses?: Status[];
    labels?: Label[];
};

const dueOptions: readonly { value: DueFilter; label: TranslationKey }[] = [
    { value: "overdue", label: "filter.due.overdue" },
    { value: "today", label: "filter.due.today" },
    { value: "week", label: "filter.due.week" },
    { value: "none", label: "filter.due.none" },
];

const doneOptions: readonly { value: DoneFilter; label: TranslationKey }[] = [
    { value: "all", label: "filter.done.all" },
    { value: "open", label: "filter.done.open" },
    { value: "done", label: "filter.done.done" },
];

function toggle<T>(list: T[], value: T): T[] {
    return list.includes(value)
        ? list.filter((item) => item !== value)
        : [...list, value];
}

export function FilterMenu({
    filters,
    onChange,
    statuses = [],
    labels = [],
}: FilterMenuProps) {
    const t = useTranslate();
    const anchorRef = useRef<HTMLDivElement>(null);
    const [open, setOpen] = useState(false);
    const active = hasActiveFilters(filters);

    return (
        <div ref={anchorRef} className="relative">
            <HeaderAction
                icon={FiFilter}
                label={t("filter.title")}
                aria-expanded={open}
                aria-haspopup="dialog"
                onClick={() => setOpen((value) => !value)}
            />
            {active && (
                <span
                    aria-hidden
                    className="absolute -top-0.5 -right-0.5 size-2 rounded-full bg-accent"
                />
            )}
            <Popover
                open={open}
                onClose={() => setOpen(false)}
                anchorRef={anchorRef}
                placement="bottom-end"
                className="max-h-[70vh] w-64 overflow-y-auto p-2"
            >
                <div role="group" aria-label={t("filter.title")}>
                    {statuses.length > 0 && (
                        <Section title={t("filter.status")}>
                            {statuses.map((status) => (
                                <CheckRow
                                    key={status.id}
                                    kind="checkbox"
                                    checked={filters.status.includes(status.id)}
                                    onClick={() =>
                                        onChange({
                                            ...filters,
                                            status: toggle(filters.status, status.id),
                                        })
                                    }
                                    icon={
                                        <span
                                            aria-hidden
                                            className="size-2.5 rounded-full"
                                            style={{ backgroundColor: status.color }}
                                        />
                                    }
                                    label={status.title}
                                />
                            ))}
                        </Section>
                    )}
                    <Section title={t("filter.priority")}>
                        {priorities.map(({ value, label, Icon, color }) => (
                            <CheckRow
                                key={value}
                                kind="checkbox"
                                checked={filters.priority.includes(value)}
                                onClick={() =>
                                    onChange({
                                        ...filters,
                                        priority: toggle<TaskPriority>(
                                            filters.priority,
                                            value,
                                        ),
                                    })
                                }
                                icon={<Icon aria-hidden className={color} />}
                                label={t(label)}
                            />
                        ))}
                    </Section>
                    {labels.length > 0 && (
                        <Section title={t("filter.labels")}>
                            {labels.map((label) => (
                                <CheckRow
                                    key={label.id}
                                    kind="checkbox"
                                    checked={filters.labels.includes(label.id)}
                                    onClick={() =>
                                        onChange({
                                            ...filters,
                                            labels: toggle(filters.labels, label.id),
                                        })
                                    }
                                    icon={
                                        <span
                                            aria-hidden
                                            className={`size-2.5 rounded-full ${labelDotClass(label.color)}`}
                                        />
                                    }
                                    label={label.name}
                                />
                            ))}
                        </Section>
                    )}
                    <Section title={t("filter.due")}>
                        {dueOptions.map(({ value, label }) => (
                            <CheckRow
                                key={value}
                                kind="radio"
                                checked={filters.due === value}
                                // Choosing the active one again clears it.
                                onClick={() =>
                                    onChange({
                                        ...filters,
                                        due: filters.due === value ? null : value,
                                    })
                                }
                                label={t(label)}
                            />
                        ))}
                    </Section>
                    <Section title={t("filter.done")}>
                        {doneOptions.map(({ value, label }) => (
                            <CheckRow
                                key={value}
                                kind="radio"
                                checked={filters.done === value}
                                onClick={() => onChange({ ...filters, done: value })}
                                label={t(label)}
                            />
                        ))}
                    </Section>
                </div>
            </Popover>
        </div>
    );
}

function Section({
    title,
    children,
}: {
    title: string;
    children: React.ReactNode;
}) {
    return (
        <div className="mb-2 last:mb-0">
            <p className="px-2 py-1 text-caption font-medium text-fg-subtle">
                {title}
            </p>
            {children}
        </div>
    );
}

function CheckRow({
    kind,
    checked,
    onClick,
    icon,
    label,
}: {
    kind: "checkbox" | "radio";
    checked: boolean;
    onClick: () => void;
    icon?: React.ReactNode;
    label: string;
}) {
    return (
        <button
            type="button"
            role={kind}
            aria-checked={checked}
            onClick={onClick}
            className="flex h-7 w-full cursor-pointer items-center gap-2 rounded-md px-2 text-left text-body-xs text-fg hover:bg-layer-1-hover focus-visible:outline-2 focus-visible:outline-accent"
        >
            {icon}
            <span className="flex-1 truncate">{label}</span>
            {checked && <FiCheck aria-hidden className="shrink-0 text-accent" />}
        </button>
    );
}
