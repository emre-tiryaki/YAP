import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createLabel,
    deleteLabel,
    listLabels,
    updateLabel,
} from "../api/client";
import type { CreateLabelInput, UpdateLabelInput } from "../types/label";

function labelsKey(projectId: string) {
    return ["projects", projectId, "labels"];
}

export function useLabels(projectId: string) {
    return useQuery({
        queryKey: labelsKey(projectId),
        queryFn: () => listLabels(projectId),
        enabled: projectId !== "",
    });
}

export function useCreateLabel(projectId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (input: CreateLabelInput) => createLabel(projectId, input),
        onSuccess: () =>
            queryClient.invalidateQueries({ queryKey: labelsKey(projectId) }),
    });
}

// Renaming or recolouring changes how tasks look, deleting also changes which
// labels they carry; the plan task lists live under the project key too.
export function useUpdateLabel(projectId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ labelId, input }: { labelId: string; input: UpdateLabelInput }) =>
            updateLabel(labelId, input),
        onSuccess: () =>
            queryClient.invalidateQueries({ queryKey: labelsKey(projectId) }),
    });
}

export function useDeleteLabel(projectId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: deleteLabel,
        onSuccess: () =>
            Promise.all([
                queryClient.invalidateQueries({ queryKey: ["projects", projectId] }),
                queryClient.invalidateQueries({ queryKey: ["tasks"] }),
            ]),
    });
}
