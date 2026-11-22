import { useRef, useState } from "react";
import { FiCheck, FiSliders } from "react-icons/fi";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { TaskPropertyKey } from "../types/task";
import type {
    DisplayOptions,
    GroupBy,
    OrderBy,
} from "../types/taskView";
import { allTaskPropertyKeys } from "../util/taskView";
import { HeaderAction } from "./HeaderAction";
import { Popover } from "./Popover";
import { Toggle } from "./Toggle";

type DisplayMenuProps = {
    display: DisplayOptions;
    onChange: (patch: Partial<DisplayOptions>) => void;
    layout: "board" | "list" | "calendar" | "table";
};

const groupOptions: readonly { value: GroupBy; label: TranslationKey }[] = [
    { value: "status", label: "display.group.status" },
    { value: "priority", label: "display.group.priority" },
    { value: "label", label: "display.group.label" },
    { value: "due", label: "display.group.due" },
    { value: "none", label: "display.group.none" },
];

const orderOptions: readonly { value: OrderBy; label: TranslationKey }[] = [
    { value: "manual", label: "display.order.manual" },
    { value: "created", label: "display.order.created" },
    { value: "updated", label: "display.order.updated" },
    { value: "due", label: "display.order.due" },
    { value: "priority", label: "display.order.priority" },
    { value: "title", label: "display.order.title" },
];

const propertyLabels: Record<TaskPropertyKey, TranslationKey> = {
    status: "task.property.status",
    priority: "task.property.priority",
    labels: "task.property.labels",
    startDate: "task.property.startDate",
    dueDate: "task.property.dueDate",
    subtasks: "display.property.subtasks",
    comments: "display.property.comments",
    createdAt: "display.property.createdAt",
    updatedAt: "display.property.updatedAt",
};

export function DisplayMenu({ display, onChange, layout }: DisplayMenuProps) {
    const t = useTranslate();
    const anchorRef = useRef<HTMLDivElement>(null);
    const [open, setOpen] = useState(false);
    const grouped = layout === "board" || layout === "list";
    // Created and updated only exist as spreadsheet columns.
    const keys = allTaskPropertyKeys.filter(
        (key) =>
            layout === "table" || (key !== "createdAt" && key !== "updatedAt"),
    );

    return (
        <div ref={anchorRef}>
            <HeaderAction
                icon={FiSliders}
                label={t("display.title")}
                aria-expanded={open}
                aria-haspopup="dialog"
                onClick={() => setOpen((value) => !value)}
            />
            <Popover
                open={open}
                onClose={() => setOpen(false)}
                anchorRef={anchorRef}
                placement="bottom-end"
                className="max-h-[70vh] w-72 overflow-y-auto p-2"
            >
                {grouped && (
                    <Section title={t("display.groupBy")}>
                        <div role="radiogroup" aria-label={t("display.groupBy")}>
                            {groupOptions.map(({ value, label }) => (
                                <RadioRow
                                    key={value}
                                    checked={display.groupBy === value}
                                    label={t(label)}
                                    onClick={() => onChange({ groupBy: value })}
                                />
                            ))}
                        </div>
                    </Section>
                )}
                <Section title={t("display.orderBy")}>
                    <div role="radiogroup" aria-label={t("display.orderBy")}>
                        {orderOptions.map(({ value, label }) => (
                            <RadioRow
                                key={value}
                                checked={display.orderBy === value}
                                label={t(label)}
                                onClick={() => onChange({ orderBy: value })}
                            />
                        ))}
                    </div>
                    {display.orderBy !== "manual" && (
                        <div
                            role="radiogroup"
                            aria-label={t("display.orderBy")}
                            className="mt-1 flex gap-1 px-2"
                        >
                            {(["asc", "desc"] as const).map((dir) => (
                                <button
                                    key={dir}
                                    type="button"
                                    role="radio"
                                    aria-checked={display.orderDir === dir}
                                    onClick={() => onChange({ orderDir: dir })}
                                    className={`h-6 flex-1 cursor-pointer rounded-md border text-caption ${
                                        display.orderDir === dir
                                            ? "border-accent text-fg"
                                            : "border-line text-fg-muted hover:bg-layer-1-hover"
                                    }`}
                                >
                                    {t(`display.dir.${dir}`)}
                                </button>
                            ))}
                        </div>
                    )}
                </Section>
                {grouped && (
                    <div className="flex items-center justify-between px-2 py-1.5">
                        <span className="text-body-xs text-fg">
                            {t("display.showEmpty")}
                        </span>
                        <Toggle
                            label={t("display.showEmpty")}
                            checked={display.showEmptyGroups}
                            onChange={(checked) =>
                                onChange({ showEmptyGroups: checked })
                            }
                        />
                    </div>
                )}
                <Section title={t("display.properties")}>
                    <div className="flex flex-wrap gap-1 px-2">
                        {keys.map((key) => {
                            const on = display.properties.includes(key);
                            return (
                                <button
                                    key={key}
                                    type="button"
                                    aria-pressed={on}
                                    onClick={() =>
                                        onChange({
                                            properties: on
                                                ? display.properties.filter(
                                                      (k) => k !== key,
                                                  )
                                                : allTaskPropertyKeys.filter(
                                                      (k) =>
                                                          k === key ||
                                                          display.properties.includes(k),
                                                  ),
                                        })
                                    }
                                    className={`h-6 cursor-pointer rounded-md border px-2 text-caption ${
                                        on
                                            ? "border-accent bg-accent/10 text-fg"
                                            : "border-line text-fg-muted hover:bg-layer-1-hover"
                                    }`}
                                >
                                    {t(propertyLabels[key])}
                                </button>
                            );
                        })}
                    </div>
                </Section>
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
        <div className="mb-2">
            <p className="px-2 py-1 text-caption font-medium text-fg-subtle">
                {title}
            </p>
            {children}
        </div>
    );
}

function RadioRow({
    checked,
    label,
    onClick,
}: {
    checked: boolean;
    label: string;
    onClick: () => void;
}) {
    return (
        <button
            type="button"
            role="radio"
            aria-checked={checked}
            onClick={onClick}
            className="flex h-7 w-full cursor-pointer items-center gap-2 rounded-md px-2 text-left text-body-xs text-fg hover:bg-layer-1-hover focus-visible:outline-2 focus-visible:outline-accent"
        >
            <span className="flex-1">{label}</span>
            {checked && <FiCheck aria-hidden className="shrink-0 text-accent" />}
        </button>
    );
}
