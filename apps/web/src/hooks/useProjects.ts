import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createProject, getProject, listPlans, listProjects } from "../api/client";

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