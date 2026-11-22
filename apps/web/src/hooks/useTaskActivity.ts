import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    addTaskRelation,
    listTaskActivity,
    listTaskRelations,
    removeTaskRelation,
} from "../api/client";
import type { RelationKind, RelationView } from "../types/taskActivity";

function activityKey(taskId: string) {
    return ["tasks", taskId, "activity"];
}

function relationsKey(taskId: string) {
    return ["tasks", taskId, "relations"];
}

export function useTaskActivity(taskId: string) {
    return useQuery({
        queryKey: activityKey(taskId),
        queryFn: () => listTaskActivity(taskId),
        enabled: taskId !== "",
    });
}

export function useTaskRelations(taskId: string) {
    return useQuery({
        queryKey: relationsKey(taskId),
        queryFn: () => listTaskRelations(taskId),
        enabled: taskId !== "",
    });
}

// How a relation read from this task is stored: which task is the first
// one, and the kind.
function stored(
    taskId: string,
    relatedId: string,
    view: RelationView,
): { first: string; second: string; kind: RelationKind } {
    switch (view) {
        case "blocked_by":
            return { first: relatedId, second: taskId, kind: "blocks" };
        case "duplicated_by":
            return { first: relatedId, second: taskId, kind: "duplicates" };
        case "duplicate_of":
            return { first: taskId, second: relatedId, kind: "duplicates" };
        case "relates":
            return { first: taskId, second: relatedId, kind: "relates" };
        default:
            return { first: taskId, second: relatedId, kind: "blocks" };
    }
}

// Both tasks show the relation (and the blocked marker on the boards), and
// both log it, so every task query is refreshed.
function useInvalidateTasks() {
    const queryClient = useQueryClient();
    return () =>
        Promise.all([
            queryClient.invalidateQueries({ queryKey: ["tasks"] }),
            queryClient.invalidateQueries({ queryKey: ["projects"] }),
        ]);
}

export function useAddRelation(taskId: string) {
    const invalidate = useInvalidateTasks();

    return useMutation({
        mutationFn: ({
            relatedId,
            view,
        }: {
            relatedId: string;
            view: RelationView;
        }) => {
            const { first, second, kind } = stored(taskId, relatedId, view);
            return addTaskRelation(first, second, kind);
        },
        onSettled: invalidate,
    });
}

export function useRemoveRelation(taskId: string) {
    const invalidate = useInvalidateTasks();

    return useMutation({
        mutationFn: (relatedId: string) => removeTaskRelation(taskId, relatedId),
        onSettled: invalidate,
    });
}
