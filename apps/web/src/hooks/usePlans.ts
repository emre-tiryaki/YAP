import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createStatus,
    getPlan,
    listStatuses,
    moveStatus,
    updateStatus,
} from "../api/client";
import type {
    CreateStatusInput,
    Status,
    UpdateStatusInput,
} from "../types/status";

export function planKey(projectId: string, planId: string) {
    return ["projects", projectId, "plans", planId];
}

function statusesKey(projectId: string, planId: string) {
    return [...planKey(projectId, planId), "statuses"];
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
        queryKey: statusesKey(projectId, planId),
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
                queryKey: statusesKey(projectId, planId),
            }),
    });
}

export function useUpdateStatus(projectId: string, planId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            statusId,
            input,
        }: {
            statusId: string;
            input: UpdateStatusInput;
        }) => updateStatus(projectId, planId, statusId, input),
        onSuccess: () =>
            queryClient.invalidateQueries({
                queryKey: statusesKey(projectId, planId),
            }),
    });
}

export type MoveStatusVariables = {
    statusId: string;
    index: number;
};

// Places the status at `index` among the other statuses. Applying it twice
// gives the same result.
function applyStatusMove(statuses: Status[], move: MoveStatusVariables) {
    const status = statuses.find((s) => s.id === move.statusId);
    if (!status) return statuses;
    const rest = statuses.filter((s) => s.id !== move.statusId);
    return [...rest.slice(0, move.index), status, ...rest.slice(move.index)];
}

export function useMoveStatus(projectId: string, planId: string) {
    const queryClient = useQueryClient();
    const key = statusesKey(projectId, planId);

    return useMutation({
        mutationFn: ({ statusId, index }: MoveStatusVariables) =>
            moveStatus(projectId, planId, statusId, index),
        onMutate: async (move) => {
            queryClient.setQueryData<Status[]>(
                key,
                (old) => old && applyStatusMove(old, move),
            );
            // An in-flight refetch would overwrite the optimistic order, so
            // cancel it and re-apply the (idempotent) move afterwards.
            await queryClient.cancelQueries({ queryKey: key });
            queryClient.setQueryData<Status[]>(
                key,
                (old) => old && applyStatusMove(old, move),
            );
        },
        // On failure the refetch restores the server's order.
        onSettled: () => queryClient.invalidateQueries({ queryKey: key }),
    });
}
