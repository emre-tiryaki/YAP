import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createPlan,
    createProject,
    deletePlan,
    deleteProject,
    getProject,
    listPlans,
    listProjects,
    updatePlan,
    updateProject,
} from "../api/client";
import type { CreatePlanInput, UpdatePlanInput } from "../types/plan";
import { planKey } from "./usePlans";
import type { UpdateProjectInput } from "../types/project";

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

// Renaming or deleting a project or plan changes names and task counts on
// every page, and deleting drops tasks from the upcoming list, so all project
// and task queries are refreshed. Queries under deletedKey (the deleted
// project or plan) are only marked stale: refetching them would 404 while the
// detail page that was showing them navigates away.
function invalidateProjectViews(
    queryClient: ReturnType<typeof useQueryClient>,
    deletedKey?: readonly unknown[],
) {
    const isDeleted = (queryKey: readonly unknown[]) =>
        deletedKey !== undefined &&
        deletedKey.every((part, i) => queryKey[i] === part);

    return Promise.all([
        queryClient.invalidateQueries({
            queryKey: ["projects"],
            predicate: (query) => !isDeleted(query.queryKey),
        }),
        queryClient.invalidateQueries({ queryKey: ["tasks"] }),
        deletedKey &&
            queryClient.invalidateQueries({
                queryKey: deletedKey,
                refetchType: "none",
            }),
    ]);
}

type UpdateProjectVariables = {
    projectId: string;
    input: UpdateProjectInput;
};

export function useUpdateProject() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ projectId, input }: UpdateProjectVariables) =>
            updateProject(projectId, input),
        onSuccess: () => invalidateProjectViews(queryClient),
    });
}

export function useDeleteProject() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: deleteProject,
        onSuccess: (_data, projectId) =>
            invalidateProjectViews(queryClient, ["projects", projectId]),
    });
}

type UpdatePlanVariables = {
    planId: string;
    input: UpdatePlanInput;
};

export function useUpdatePlan(projectId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ planId, input }: UpdatePlanVariables) =>
            updatePlan(projectId, planId, input),
        onSuccess: () => invalidateProjectViews(queryClient),
    });
}

export function useDeletePlan(projectId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (planId: string) => deletePlan(projectId, planId),
        onSuccess: (_data, planId) =>
            invalidateProjectViews(queryClient, planKey(projectId, planId)),
    });
}
