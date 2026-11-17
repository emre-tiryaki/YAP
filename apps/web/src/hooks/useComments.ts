import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    createComment,
    deleteComment,
    listComments,
    listRecentComments,
    updateComment,
} from "../api/client";
import { readCommentsSince, writeCommentsSince } from "../util/notifications";

function commentsKey(taskId: string) {
    return ["tasks", taskId, "comments"];
}

export function useComments(taskId: string) {
    return useQuery({
        queryKey: commentsKey(taskId),
        queryFn: () => listComments(taskId),
        enabled: taskId !== "",
    });
}

// Comments other members wrote since the comment notifications were turned
// on (or since the last one seen), polled while `enabled`.
export function useRecentComments(enabled: boolean, refetchIntervalMs: number) {
    return useQuery({
        queryKey: ["comments", "recent"],
        queryFn: () => {
            let since = readCommentsSince();
            if (!since) {
                since = new Date();
                writeCommentsSince(since);
            }
            return listRecentComments(since);
        },
        enabled,
        refetchInterval: refetchIntervalMs,
    });
}

// The board cards show a comment count, so plan task lists (nested under
// "projects") are refreshed along with the task's comments.
function useInvalidateComments(taskId: string) {
    const queryClient = useQueryClient();
    return () =>
        Promise.all([
            queryClient.invalidateQueries({ queryKey: commentsKey(taskId) }),
            queryClient.invalidateQueries({ queryKey: ["projects"] }),
        ]);
}

export function useCreateComment(taskId: string) {
    const invalidate = useInvalidateComments(taskId);
    return useMutation({
        mutationFn: (body: string) => createComment(taskId, body),
        onSuccess: invalidate,
    });
}

type UpdateCommentVariables = {
    commentId: string;
    body: string;
};

export function useUpdateComment(taskId: string) {
    const invalidate = useInvalidateComments(taskId);
    return useMutation({
        mutationFn: ({ commentId, body }: UpdateCommentVariables) =>
            updateComment(commentId, body),
        onSuccess: invalidate,
    });
}

export function useDeleteComment(taskId: string) {
    const invalidate = useInvalidateComments(taskId);
    return useMutation({
        mutationFn: deleteComment,
        onSuccess: invalidate,
    });
}
