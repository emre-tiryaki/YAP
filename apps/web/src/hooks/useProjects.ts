import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createPlan,
    createProject,
    getProject,
    listPlans,
    listProjects,
} from "../api/client";
import type { CreatePlanInput } from "../types/plan";

export function useProjects() {
    return useQuery({ queryKey: ["projects"], queryFn: listProjects });
}

export function useCreateProjects() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: createProject,
        onSuccess: () =>
            queryClient.invalidateQueries({ queryKey: ["projects"] }),
    });
}

export function useProject(projectId: string) {
    return useQuery({
        queryKey: ["projects", projectId],
        queryFn: () => getProject(projectId),
        enabled: projectId !== "",
    });
}

export function useProjectPlans(projectId: string) {
    return useQuery({
        queryKey: ["projects", projectId, "plans"],
        queryFn: () => listPlans(projectId),
        enabled: projectId !== "",
    });
}

export function useCreatePlan(projectId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (input: CreatePlanInput) => createPlan(projectId, input),
        onSuccess: () =>
            queryClient.invalidateQueries({
                queryKey: ["projects", projectId, "plans"],
            }),
    });
}
