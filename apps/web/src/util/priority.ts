import type { IconType } from "react-icons";
import {
    TbAlertSquareFilled,
    TbAntennaBars1,
    TbAntennaBars2,
    TbAntennaBars4,
    TbAntennaBars5,
} from "react-icons/tb";
import type { TranslationKey } from "../hooks/useTranslate";
import type { TaskPriority } from "../types/task";

type PriorityInfo = {
    value: TaskPriority;
    label: TranslationKey;
    Icon: IconType;
    // Text colour of the icon.
    color: string;
};

// Most urgent first. Every priority has its own icon, so none relies on
// colour alone.
export const priorities: readonly PriorityInfo[] = [
    { value: "urgent", label: "task.priority.urgent", Icon: TbAlertSquareFilled, color: "text-red-600" },
    { value: "high", label: "task.priority.high", Icon: TbAntennaBars5, color: "text-orange-600" },
    { value: "medium", label: "task.priority.medium", Icon: TbAntennaBars4, color: "text-amber-600" },
    { value: "low", label: "task.priority.low", Icon: TbAntennaBars2, color: "text-blue-600" },
    { value: "none", label: "task.priority.none", Icon: TbAntennaBars1, color: "text-fg-placeholder" },
];

export function priorityInfo(value: TaskPriority): PriorityInfo {
    return priorities.find((p) => p.value === value) ?? priorities[priorities.length - 1];
}
