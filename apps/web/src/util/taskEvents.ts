import type { TranslationKey } from "../hooks/useTranslate";
import type { TaskPriority } from "../types/task";
import type { RelationView, TaskEvent } from "../types/taskActivity";
import { priorityInfo } from "./priority";

type Translate = (key: TranslationKey) => string;

const priorityValues: readonly string[] = [
    "urgent",
    "high",
    "medium",
    "low",
    "none",
];

// The sentence of an activity entry, e.g. "Status changed from Todo to Doing".
// `formatDate` renders the RFC 3339 dates the server logs.
export function describeEvent(
    event: TaskEvent,
    t: Translate,
    formatDate: (date: Date) => string,
): string {
    const none = t("task.event.none");
    const d = event.details;
    const date = (value: string | undefined) =>
        value ? formatDate(new Date(value)) : none;
    const priority = (value: string | undefined) =>
        value && priorityValues.includes(value)
            ? t(priorityInfo(value as TaskPriority).label)
            : none;
    const text = (value: string | undefined) => (value ? value : none);

    const fill = (key: TranslationKey, values: Record<string, string>) =>
        Object.entries(values).reduce(
            (out, [name, value]) => out.replace(`{${name}}`, value),
            t(key),
        );

    switch (event.action) {
        case "renamed":
            return fill("task.event.renamed", {
                from: d.from ?? "",
                to: d.to ?? "",
            });
        case "status_changed":
            return fill("task.event.status_changed", {
                from: text(d.from),
                to: text(d.to),
            });
        case "priority_changed":
            return fill("task.event.priority_changed", {
                from: priority(d.from),
                to: priority(d.to),
            });
        case "due_changed":
            return fill("task.event.due_changed", {
                from: date(d.from),
                to: date(d.to),
            });
        case "start_changed":
            return fill("task.event.start_changed", {
                from: date(d.from),
                to: date(d.to),
            });
        case "labels_changed":
            return fill("task.event.labels_changed", {
                from: text(d.from),
                to: text(d.to),
            });
        case "relation_added":
        case "relation_removed":
            return fill(
                event.action === "relation_added"
                    ? "task.event.relation_added"
                    : "task.event.relation_removed",
                {
                    kind: t(`task.relation.${(d.kind ?? "relates") as RelationView}`),
                    task: d.task ?? "",
                },
            );
        default:
            return t(`task.event.${event.action}`);
    }
}
