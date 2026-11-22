import {
    keepPreviousData,
    useMutation,
    useQuery,
    useQueryClient,
} from "@tanstack/react-query";
import { useEffect, useState } from "react";
import {
    ApiError,
    createAiChat,
    createAiConnection,
    deleteAiChat,
    deleteAiConnection,
    getAiChat,
    getAiStatus,
    listAiChats,
    listAiConnections,
    listAiModels,
    regenerateAiMessage,
    renameAiChat,
    sendAiMessage,
    stopAiChat,
    streamAiChat,
    testAiConnection,
    updateAiConnection,
} from "../api/client";
import type { UpdateAiConnectionInput } from "../types/ai";
import type { TranslationKey } from "./useTranslate";

const connectionsKey = ["ai", "connections"] as const;
const chatsKey = ["ai", "chats"] as const;
const chatKey = ["ai", "chat"] as const;
const statusKey = ["ai", "status"] as const;

const POLL_MS = 2_000;

// Maps a backend error code to the text shown for it.
export function aiErrorCodeKey(code: string | null): TranslationKey {
    switch (code) {
        case "ai_busy":
            return "ai.error.busy";
        case "ai_connection_missing":
            return "ai.error.connectionMissing";
        case "ai_key_missing":
            return "ai.error.keyMissing";
        case "ai_unreachable":
            return "ai.error.unreachable";
        case "ai_auth_failed":
            return "ai.error.authFailed";
        case "ai_rate_limited":
            return "ai.error.rateLimited";
        case "ai_model_not_found":
            return "ai.error.modelNotFound";
        case "ai_rejected":
            return "ai.error.rejected";
        case "keychain_unavailable":
            return "resources.error.keychain";
        case "invalid_input":
            return "ai.error.invalid";
        case "not_found":
            return "ai.error.notFound";
        default:
            return "common.error";
    }
}

export function aiErrorKey(error: unknown): TranslationKey {
    return error instanceof ApiError
        ? aiErrorCodeKey(error.message)
        : "common.error";
}

export function useAiConnections() {
    return useQuery({ queryKey: connectionsKey, queryFn: listAiConnections });
}

function useInvalidateConnections() {
    const queryClient = useQueryClient();
    return () => queryClient.invalidateQueries({ queryKey: connectionsKey });
}

export function useCreateAiConnection() {
    const invalidate = useInvalidateConnections();
    return useMutation({
        mutationFn: createAiConnection,
        onSuccess: invalidate,
    });
}

export function useUpdateAiConnection() {
    const invalidate = useInvalidateConnections();
    return useMutation({
        mutationFn: (v: { id: string; input: UpdateAiConnectionInput }) =>
            updateAiConnection(v.id, v.input),
        onSuccess: invalidate,
    });
}

export function useDeleteAiConnection() {
    const invalidate = useInvalidateConnections();
    return useMutation({
        mutationFn: deleteAiConnection,
        onSuccess: invalidate,
    });
}

// Lists the models of the settings in a connection form. Without a new key
// the saved connection is asked, so its stored key is used.
export function useTestAiConnection(saved: { id: string } | null) {
    return useMutation({
        mutationFn: (input: Parameters<typeof testAiConnection>[0]) =>
            saved && !input.apiKey
                ? listAiModels(saved.id)
                : testAiConnection(input),
    });
}

// The models of a saved connection, asked from its server when needed.
export function useAiModels(connectionId: string, enabled: boolean) {
    return useQuery({
        queryKey: [...connectionsKey, connectionId, "models"],
        queryFn: () => listAiModels(connectionId),
        enabled: enabled && connectionId !== "",
        staleTime: 60_000,
        retry: false,
    });
}

export function useAiChats(query: string) {
    return useQuery({
        queryKey: [...chatsKey, query],
        queryFn: ({ signal }) => listAiChats(query, signal),
        placeholderData: keepPreviousData,
        refetchInterval: (q) =>
            q.state.data?.some((chat) => chat.generating) ? POLL_MS : false,
    });
}

export function useAiChat(chatId: string) {
    return useQuery({
        queryKey: [...chatKey, chatId],
        queryFn: () => getAiChat(chatId),
        enabled: chatId !== "",
        retry: false,
        refetchInterval: (q) =>
            q.state.data?.messages.some((m) => m.status === "streaming")
                ? POLL_MS
                : false,
    });
}

// The chat being answered, or null. Polled while one is, so the composer of
// another chat learns when it is free again.
export function useAiStatus(): { activeChatId: string | null } {
    const { data } = useQuery({
        queryKey: statusKey,
        queryFn: getAiStatus,
        refetchInterval: (q) => (q.state.data ? POLL_MS : false),
    });
    return { activeChatId: data ?? null };
}

function useInvalidateChats() {
    const queryClient = useQueryClient();
    return () =>
        Promise.all([
            queryClient.invalidateQueries({ queryKey: chatsKey }),
            queryClient.invalidateQueries({ queryKey: chatKey }),
            queryClient.invalidateQueries({ queryKey: statusKey }),
        ]);
}

export function useCreateAiChat() {
    const invalidate = useInvalidateChats();
    return useMutation({ mutationFn: createAiChat, onSuccess: invalidate });
}

export function useSendAiMessage(chatId: string) {
    const invalidate = useInvalidateChats();
    return useMutation({
        mutationFn: (input: Parameters<typeof sendAiMessage>[1]) =>
            sendAiMessage(chatId, input),
        onSuccess: invalidate,
    });
}

export function useRegenerateAiMessage(chatId: string) {
    const invalidate = useInvalidateChats();
    return useMutation({
        mutationFn: (input: { connectionId: string; model: string }) =>
            regenerateAiMessage(chatId, input),
        onSuccess: invalidate,
    });
}

export function useStopAiChat() {
    const invalidate = useInvalidateChats();
    return useMutation({ mutationFn: stopAiChat, onSuccess: invalidate });
}

export function useRenameAiChat() {
    const invalidate = useInvalidateChats();
    return useMutation({
        mutationFn: (v: { chatId: string; title: string }) =>
            renameAiChat(v.chatId, v.title),
        onSuccess: invalidate,
    });
}

export function useDeleteAiChat() {
    const invalidate = useInvalidateChats();
    return useMutation({ mutationFn: deleteAiChat, onSuccess: invalidate });
}

type StreamState = { chatId: string; messageId: string; content: string };

// Follows the answer being written for a chat. The stream is a window onto a
// generation that runs on the server: closing it (leaving the page) does not
// stop the answer. Returns null until the first snapshot arrives.
export function useAiStream(chatId: string, enabled: boolean) {
    const queryClient = useQueryClient();
    const [state, setState] = useState<StreamState | null>(null);

    useEffect(() => {
        if (!enabled || chatId === "") return;
        const controller = new AbortController();
        const refresh = () => {
            void queryClient.invalidateQueries({ queryKey: chatsKey });
            void queryClient.invalidateQueries({ queryKey: chatKey });
            void queryClient.invalidateQueries({ queryKey: statusKey });
        };
        streamAiChat(
            chatId,
            {
                onSnapshot: (messageId, content) =>
                    setState({ chatId, messageId, content }),
                onDelta: (text) =>
                    setState((prev) =>
                        prev && prev.chatId === chatId
                            ? { ...prev, content: prev.content + text }
                            : prev,
                    ),
                onDone: refresh,
            },
            controller.signal,
        )
            .then(refresh)
            .catch(() => {
                // Aborted on leaving, or the server is gone; the chat query
                // keeps polling while a message is streaming.
            });
        return () => controller.abort();
    }, [chatId, enabled, queryClient]);

    return state && state.chatId === chatId ? state : null;
}
