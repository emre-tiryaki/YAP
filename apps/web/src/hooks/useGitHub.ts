import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    addRepository,
    COMMITS_PER_PAGE,
    listRepoCommits,
    cloneRepository,
    forgetRepositoryClone,
    listLocalFolders,
    listRepositoryClones,
    disconnectGitHub,
    getGitHubStatus,
    getRepository,
    listGitHubRepos,
    listRepoEntries,
    pollGitHubConnect,
    readRepoText,
    startGitHubConnect,
    syncRepository,
} from "../api/client";
import type { Repository } from "../types/github";

const githubKey = ["github"] as const;
// Repository folders are resources: their queries live under the resource
// key, so changes to the tree refresh them too.
const resourcesKey = ["resources"] as const;

export function useGitHubStatus() {
    return useQuery({ queryKey: [...githubKey, "status"], queryFn: getGitHubStatus });
}

export function useStartGitHubConnect() {
    return useMutation({ mutationFn: startGitHubConnect });
}

export function usePollGitHubConnect() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: pollGitHubConnect,
        onSuccess: (result) => {
            if (result.state === "connected") void queryClient.invalidateQueries({ queryKey: githubKey });
        },
    });
}

export function useDisconnectGitHub() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: disconnectGitHub,
        onSuccess: () => queryClient.invalidateQueries({ queryKey: githubKey }),
    });
}

export function useGitHubRepos(enabled: boolean) {
    return useQuery({
        queryKey: [...githubKey, "repos"],
        queryFn: listGitHubRepos,
        enabled,
        staleTime: 60_000,
        retry: false,
    });
}

export function useAddRepository() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: addRepository,
        onSettled: () => queryClient.invalidateQueries({ queryKey: resourcesKey }),
    });
}

// While a copy is being fetched the state is polled, so the page shows the
// files as soon as they are there; otherwise it is checked now and then to
// pick up background updates.
function refetchInterval(repo: Repository | undefined): number {
    if (!repo) return 0;
    if (repo.state === "fetching" || (repo.state === "pending" && repo.error === null)) return 2_000;
    return 30_000;
}

export function useRepository(resourceId: string) {
    return useQuery({
        queryKey: [...resourcesKey, "repository", resourceId],
        queryFn: () => getRepository(resourceId),
        enabled: resourceId !== "",
        refetchInterval: (query) => refetchInterval(query.state.data),
    });
}

export function useSyncRepository() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: syncRepository,
        onSuccess: (repo) =>
            queryClient.setQueryData([...resourcesKey, "repository", repo.resourceId], repo),
    });
}

// Entries and texts are keyed by the commit, so an update is a new key.
export function useRepoEntries(resourceId: string, path: string, commit: string | null) {
    return useQuery({
        queryKey: [...resourcesKey, "repository", resourceId, "entries", commit ?? "", path],
        queryFn: () => listRepoEntries(resourceId, path),
        enabled: commit !== null,
        staleTime: Infinity,
        retry: false,
    });
}

export function useRepoText(resourceId: string, path: string, commit: string) {
    return useQuery({
        queryKey: [...resourcesKey, "repository", resourceId, "text", commit, path],
        queryFn: () => readRepoText(resourceId, path, commit),
        staleTime: Infinity,
        retry: false,
    });
}

export function useRepositoryClones(resourceId: string) {
    return useQuery({
        queryKey: [...resourcesKey, "repository", resourceId, "clones"],
        queryFn: () => listRepositoryClones(resourceId),
        enabled: resourceId !== "",
    });
}

export function useCloneRepository() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: cloneRepository,
        onSuccess: (_clone, input) =>
            queryClient.invalidateQueries({ queryKey: [...resourcesKey, "repository", input.resourceId, "clones"] }),
    });
}

export function useForgetRepositoryClone(resourceId: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (cloneId: string) => forgetRepositoryClone(resourceId, cloneId),
        onSuccess: () =>
            queryClient.invalidateQueries({ queryKey: [...resourcesKey, "repository", resourceId, "clones"] }),
    });
}

// The folders on this computer are not cached: they change outside YAP.
export function useLocalFolders(path: string, enabled: boolean) {
    return useQuery({
        queryKey: ["local-folders", path],
        queryFn: () => listLocalFolders(path),
        enabled,
        staleTime: 0,
        gcTime: 0,
        retry: false,
    });
}

// The branch history comes from GitHub page by page. Keyed by the copy's
// commit, so an update shows the new commits.
export function useRepoCommits(resourceId: string, commit: string | null) {
    return useInfiniteQuery({
        queryKey: [...resourcesKey, "repository", resourceId, "commits", commit ?? ""],
        queryFn: ({ pageParam }) => listRepoCommits(resourceId, pageParam),
        initialPageParam: 1,
        getNextPageParam: (lastPage, pages) => (lastPage.length < COMMITS_PER_PAGE ? undefined : pages.length + 1),
        retry: false,
        staleTime: 60_000,
    });
}
