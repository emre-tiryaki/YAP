import { useSearch } from "./useSearch";
import type { AiAttachmentKind } from "../types/ai";

export type AiMentionHit = {
    key: string;
    kind: AiAttachmentKind;
    targetId: string;
    label: string;
    detail: string;
};

const maxHits = 8;

// What an "@query" in the composer can attach: tasks, plans, projects and
// text files from the workspace search.
export function useAiMentions(query: string | null): {
    hits: AiMentionHit[];
    isPending: boolean;
} {
    const search = useSearch(query ?? "");
    if (query === null || query.trim() === "" || !search.data) {
        return { hits: [], isPending: query !== null && query.trim() !== "" && search.isPending };
    }
    const r = search.data;
    const hits: AiMentionHit[] = [
        ...r.tasks.map((task) => ({
            key: `task:${task.id}`,
            kind: "task" as const,
            targetId: task.id,
            label: task.title,
            detail: `${task.projectName} / ${task.planName}`,
        })),
        ...r.plans.map((plan) => ({
            key: `plan:${plan.id}`,
            kind: "plan" as const,
            targetId: plan.id,
            label: plan.name,
            detail: plan.projectName,
        })),
        ...r.projects.map((project) => ({
            key: `project:${project.id}`,
            kind: "project" as const,
            targetId: project.id,
            label: project.name,
            detail: "",
        })),
        ...r.resources
            .filter((res) => res.kind === "file" && res.isText)
            .map((res) => ({
                key: `resource:${res.id}`,
                kind: "resource" as const,
                targetId: res.id,
                label: res.name,
                detail: res.parentName ?? "",
            })),
    ].slice(0, maxHits);
    return { hits, isPending: false };
}
