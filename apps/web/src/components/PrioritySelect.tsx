import { useTranslate } from "../hooks/useTranslate";
import type { TaskPriority } from "../types/task";
import { priorities, priorityInfo } from "../util/priority";
import { PropertySelect } from "./PropertySelect";

type PrioritySelectProps = {
    value: TaskPriority;
    onChange: (priority: TaskPriority) => void;
    variant: "pill" | "ghost" | "cell";
    // Shows only the icon (the label stays as the accessible name).
    iconOnly?: boolean;
};

// A PropertySelect over the priorities, each with its icon.
export function PrioritySelect({ value, onChange, variant, iconOnly = false }: PrioritySelectProps) {
    const t = useTranslate();
    const current = priorityInfo(value);

    return (
        <PropertySelect
            variant={variant}
            ariaLabel={t("task.property.priority")}
            tooltip={t(current.label)}
            placeholder={t("task.property.priority")}
            value={value}
            triggerLabel={iconOnly ? <span className="sr-only">{t(current.label)}</span> : t(current.label)}
            onChange={(next) =>
                typeof next === "string" && onChange(next as TaskPriority)
            }
            options={priorities.map(({ value: v, label, Icon, color }) => ({
                value: v,
                label: t(label),
                icon: <Icon aria-hidden className={color} />,
            }))}
        />
    );
}
