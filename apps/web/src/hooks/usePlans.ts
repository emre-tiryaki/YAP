import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createStatus, getPlan, listStatuses } from "../api/client";
import type { CreateStatusInput } from "../types/status";

export function planKey(projectId: string, planId: string) {
    return ["projects", projectId, "plans", planId];
}

export function usePlan(projectId: string, planId: string) {
    return useQuery({
        queryKey: planKey(projectId, planId),
        queryFn: () => getPlan(projectId, planId),
        enabled: projectId !== "" && planId !== "",
    });
}

export function usePlanStatuses(projectId: string, planId: string) {
    return useQuery({
        queryKey: [...planKey(projectId, planId), "statuses"],
        queryFn: () => listStatuses(projectId, planId),
        enabled: projectId !== "" && planId !== "",
    });
}

export function useCreateStatus(projectId: string, planId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (input: CreateStatusInput) =>
            createStatus(projectId, planId, input),
        onSuccess: () =>
            queryClient.invalidateQueries({
                queryKey: [...planKey(projectId, planId), "statuses"],
            }),
    });
}
