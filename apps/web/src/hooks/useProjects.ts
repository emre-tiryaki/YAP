import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createProject, listProjects } from "../api/client";

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
