import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { CreateTaskInput } from "../types/task";
import { createTask, listUpcomingTasks } from "../api/client";

type CreateTaskVariables = {
    projectId: string;
    input: CreateTaskInput;
};

export function useUpcomingTasks() {
    return useQuery({
        queryKey: ["tasks", "upcoming"],
        queryFn: listUpcomingTasks,
    });
}

export function useCreateTask() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ projectId, input }: CreateTaskVariables) =>
            createTask(projectId, input),
        onSuccess: () =>
            Promise.all([
                queryClient.invalidateQueries({ queryKey: ["tasks"] }),
                queryClient.invalidateQueries({ queryKey: ["projects"] }),
            ]),
    });
}
