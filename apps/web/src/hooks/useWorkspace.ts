import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getWorkspace, updateWorkspace } from "../api/client";

export function useWorkspace() {
    return useQuery({queryKey: ['workspace'], queryFn: getWorkspace})
}

export function useUpdateWorkspace() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: updateWorkspace,
        onSuccess: (workspace) => {
            queryClient.setQueryData(["workspace"], workspace);
            // Comments show the author's current name.
            return queryClient.invalidateQueries({
                predicate: (query) => query.queryKey[2] === "comments",
            });
        },
    });
}
