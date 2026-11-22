import { FiTag } from "react-icons/fi";
import { useCreateLabel, useLabels } from "../hooks/useLabels";
import { useTranslate } from "../hooks/useTranslate";
import type { LabelColor } from "../types/label";
import { labelDotClass } from "../util/labelColors";
import { PropertySelect } from "./PropertySelect";

type LabelsSelectProps = {
    projectId: string;
    value: string[];
    onChange: (labelIds: string[]) => void;
    variant: "pill" | "ghost" | "cell";
};

function Dot({ color }: { color: LabelColor }) {
    return (
        <span
            aria-hidden
            className={`size-2.5 shrink-0 rounded-full ${labelDotClass(color)}`}
        />
    );
}

// A multi select over the labels of a project. Typing a name that does not
// exist offers to create it, and adds it to the task.
export function LabelsSelect({ projectId, value, onChange, variant }: LabelsSelectProps) {
    const t = useTranslate();
    const labels = useLabels(projectId);
    const create = useCreateLabel(projectId);
    const all = labels.data ?? [];
    const chosen = value.flatMap((id) => {
        const label = all.find((l) => l.id === id);
        return label ? [label] : [];
    });

    // Up to two labels by name, then +N.
    const shown = chosen.slice(0, 2);
    const trigger =
        chosen.length === 0 ? (
            <span className="flex items-center gap-1">
                <FiTag aria-hidden />
                {variant !== "pill" && (
                    <span className="text-fg-placeholder">{t("label.none")}</span>
                )}
            </span>
        ) : (
            <span className="flex items-center gap-2">
                {shown.map((label) => (
                    <span key={label.id} className="flex items-center gap-1">
                        <Dot color={label.color} />
                        {label.name}
                    </span>
                ))}
                {chosen.length > shown.length && (
                    <span className="text-fg-subtle">+{chosen.length - shown.length}</span>
                )}
            </span>
        );

    return (
        <PropertySelect
            multiple
            variant={variant}
            ariaLabel={t("task.property.labels")}
            tooltip={t("task.property.labels")}
            placeholder={t("task.property.labels")}
            value={value}
            triggerLabel={trigger}
            onChange={(next) => Array.isArray(next) && onChange(next)}
            options={all.map((label) => ({
                value: label.id,
                label: label.name,
                icon: <Dot color={label.color} />,
            }))}
            create={{
                label: (name) => t("label.create").replace("{name}", name),
                onCreate: (name) =>
                    create.mutate(
                        { name, color: "gray" },
                        { onSuccess: (label) => onChange([...value, label.id]) },
                    ),
            }}
        />
    );
}
