import {
    useInfiniteQuery,
    useMutation,
    useQuery,
    useQueryClient,
} from "@tanstack/react-query";
import {
    copyResource,
    createFile,
    createFolder,
    emptyTrash,
    getResource,
    linkResource,
    listResourceEvents,
    listResourceTasks,
    listResources,
    listTaskResources,
    listTrash,
    moveResource,
    purgeResource,
    readResourceText,
    renameResource,
    restoreResource,
    saveResourceText,
    trashResource,
    unlinkResource,
    uploadFile,
} from "../api/client";
import type {
    OnConflict,
    Resource,
    ResourceEventScope,
} from "../types/resource";

// Everything resource-related lives under one key, so any change refreshes
// the tree, the trash, task links and the activity logs together.
const resourcesKey = ["resources"] as const;
const EVENT_PAGE = 30;

export function useResources() {
    return useQuery({
        queryKey: [...resourcesKey, "list"],
        queryFn: listResources,
    });
}

// One resource, trashed ones included; for items not in the live tree.
export function useResource(id: string, enabled: boolean) {
    return useQuery({
        queryKey: [...resourcesKey, "item", id],
        queryFn: () => getResource(id),
        enabled: enabled && id !== "",
        retry: false,
    });
}

export function useTrash() {
    return useQuery({
        queryKey: [...resourcesKey, "trash"],
        queryFn: listTrash,
    });
}

export function useResourceText(resource: Resource | undefined) {
    return useQuery({
        queryKey: [
            ...resourcesKey,
            "text",
            resource?.id ?? "",
            resource?.revision ?? 0,
        ],
        queryFn: () => readResourceText(resource!),
        enabled: resource !== undefined,
        // The editor owns the text once loaded; refetching would clobber it.
        staleTime: Infinity,
        gcTime: 0,
    });
}

export function useResourceEvents(scope: ResourceEventScope) {
    const id = scope.kind === "workspace" ? "" : scope.id;
    return useInfiniteQuery({
        queryKey: [...resourcesKey, "events", scope.kind, id],
        queryFn: ({ pageParam }) =>
            listResourceEvents(scope, pageParam, EVENT_PAGE),
        initialPageParam: null as string | null,
        getNextPageParam: (lastPage) =>
            lastPage.length < EVENT_PAGE ? undefined : lastPage.at(-1)!.id,
    });
}

export function useResourceTasks(id: string) {
    return useQuery({
        queryKey: [...resourcesKey, "tasks", id],
        queryFn: () => listResourceTasks(id),
        enabled: id !== "",
    });
}

export function useTaskResources(taskId: string) {
    return useQuery({
        queryKey: [...resourcesKey, "task", taskId],
        queryFn: () => listTaskResources(taskId),
        enabled: taskId !== "",
    });
}

function useInvalidateResources() {
    const queryClient = useQueryClient();
    return () =>
        queryClient.invalidateQueries({
            queryKey: resourcesKey,
            // The open editor's text query is keyed by revision; a new
            // revision is a new key, so the old one need not refetch.
            predicate: (query) => query.queryKey[1] !== "text",
        });
}

function useResourceMutation<TVariables, TResult>(
    mutationFn: (variables: TVariables) => Promise<TResult>,
) {
    const invalidate = useInvalidateResources();
    return useMutation({ mutationFn, onSettled: invalidate });
}

type CreateVariables = { parentId: string | null; name: string };

export function useCreateFolder() {
    return useResourceMutation(({ parentId, name }: CreateVariables) =>
        createFolder(parentId, name),
    );
}

export function useCreateFile() {
    return useResourceMutation(({ parentId, name }: CreateVariables) =>
        createFile(parentId, name),
    );
}

type UploadVariables = {
    parentId: string | null;
    file: File;
    onConflict: OnConflict;
};

export function useUploadFile() {
    return useResourceMutation(({ parentId, file, onConflict }: UploadVariables) =>
        uploadFile(parentId, file, onConflict),
    );
}

export function useRenameResource() {
    return useResourceMutation(({ id, name }: { id: string; name: string }) =>
        renameResource(id, name),
    );
}

type TransferVariables = {
    id: string;
    parentId: string | null;
    onConflict: OnConflict;
};

export function useMoveResource() {
    return useResourceMutation(({ id, parentId, onConflict }: TransferVariables) =>
        moveResource(id, parentId, onConflict),
    );
}

export function useCopyResource() {
    return useResourceMutation(({ id, parentId, onConflict }: TransferVariables) =>
        copyResource(id, parentId, onConflict),
    );
}

export function useTrashResource() {
    return useResourceMutation((id: string) => trashResource(id));
}

export function useRestoreResource() {
    return useResourceMutation((id: string) => restoreResource(id));
}

export function usePurgeResource() {
    return useResourceMutation((id: string) => purgeResource(id));
}

export function useEmptyTrash() {
    return useResourceMutation(() => emptyTrash());
}

type SaveVariables = { id: string; revision: number; text: string };

// Saving does not refetch the tree right away: the editor updates its
// revision from the response, and the list refreshes in the background.
export function useSaveResourceText() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({ id, revision, text }: SaveVariables) =>
            saveResourceText(id, revision, text),
        onSuccess: (saved) => {
            queryClient.setQueryData<Resource[]>(
                [...resourcesKey, "list"],
                (list) => list?.map((r) => (r.id === saved.id ? saved : r)),
            );
            void queryClient.invalidateQueries({
                queryKey: [...resourcesKey, "events"],
            });
        },
    });
}

type LinkVariables = { taskId: string; resourceId: string };

export function useLinkResource() {
    return useResourceMutation(({ taskId, resourceId }: LinkVariables) =>
        linkResource(taskId, resourceId),
    );
}

export function useUnlinkResource() {
    return useResourceMutation(({ taskId, resourceId }: LinkVariables) =>
        unlinkResource(taskId, resourceId),
    );
}
