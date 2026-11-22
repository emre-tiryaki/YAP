import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { FiLoader, FiMoreHorizontal, FiPlus, FiSearch } from "react-icons/fi";
import {
    aiErrorKey,
    useAiChats,
    useDeleteAiChat,
    useRenameAiChat,
} from "../hooks/useAi";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { useTranslate } from "../hooks/useTranslate";
import type { AiChat } from "../types/ai";
import { groupAiChats } from "../util/aiChatGroups";
import { Button } from "./Button";
import { ConfirmDialog } from "./ConfirmDialog";
import { InlineError } from "./InlineError";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";
import { RenameDialog } from "./RenameDialog";
import { Skeleton } from "./Skeleton";

export function AiChatList({ activeChatId }: { activeChatId: string | null }) {
    const t = useTranslate();
    const navigate = useNavigate();
    const [search, setSearch] = useState("");
    const query = useDebouncedValue(search.trim(), 250);
    const chats = useAiChats(query);
    const rename = useRenameAiChat();
    const remove = useDeleteAiChat();
    const [renaming, setRenaming] = useState<AiChat | null>(null);
    const [deleting, setDeleting] = useState<AiChat | null>(null);
    const groups = groupAiChats(chats.data ?? []);

    return (
        <aside
            aria-label={t("assistant.list.label")}
            className="flex w-64 shrink-0 flex-col gap-2 border-r border-line p-2"
        >
            <Button
                variant="primary"
                size="lg"
                leadingIcon={FiPlus}
                onClick={() => void navigate("/assistant")}
            >
                {t("assistant.new")}
            </Button>
            <label className="relative">
                <FiSearch
                    aria-hidden
                    className="absolute top-1/2 left-2 -translate-y-1/2 text-fg-placeholder"
                />
                <input
                    type="search"
                    value={search}
                    aria-label={t("assistant.search")}
                    placeholder={t("assistant.search.placeholder")}
                    onChange={(event) => setSearch(event.target.value)}
                    className="h-8 w-full rounded-md border border-line bg-layer-2 pr-2 pl-7 text-body-xs"
                />
            </label>
            <div className="min-h-0 flex-1 overflow-y-auto">
                {chats.isPending ? (
                    <div className="flex flex-col gap-1.5 p-1">
                        <Skeleton className="h-7" />
                        <Skeleton className="h-7" />
                        <Skeleton className="h-7" />
                    </div>
                ) : chats.isError ? (
                    <InlineError
                        message={t("common.error")}
                        onRetry={() => void chats.refetch()}
                    />
                ) : groups.length === 0 ? (
                    <p className="p-2 text-body-xs text-fg-subtle">
                        {t(query ? "assistant.list.noMatch" : "assistant.list.empty")}
                    </p>
                ) : (
                    groups.map((group) => (
                        <section key={group.titleKey} className="mb-2">
                            <h2 className="px-2 py-1 text-caption font-medium text-fg-muted">
                                {t(group.titleKey)}
                            </h2>
                            <ul>
                                {group.chats.map((chat) => (
                                    <li
                                        key={chat.id}
                                        className={`group flex items-center rounded-md hover:bg-layer-1-hover ${
                                            chat.id === activeChatId ? "bg-layer-1-hover" : ""
                                        }`}
                                    >
                                        <Link
                                            to={`/assistant/${chat.id}`}
                                            aria-current={chat.id === activeChatId ? "page" : undefined}
                                            className="flex h-8 min-w-0 flex-1 items-center gap-1.5 px-2 text-body-xs text-fg"
                                        >
                                            <span className="truncate">{chat.title}</span>
                                            {chat.generating && (
                                                <FiLoader
                                                    aria-label={t("assistant.chat.generating")}
                                                    className="shrink-0 animate-spin text-fg-muted"
                                                />
                                            )}
                                        </Link>
                                        <Menu
                                            placement="bottom-end"
                                            trigger={(props) => (
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    className="mr-1 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100"
                                                    aria-label={`${t("assistant.chat.menu")}: ${chat.title}`}
                                                    {...props}
                                                >
                                                    <FiMoreHorizontal aria-hidden />
                                                </Button>
                                            )}
                                        >
                                            <MenuItem
                                                label={t("assistant.chat.rename")}
                                                onSelect={() => setRenaming(chat)}
                                            />
                                            <MenuItem
                                                danger
                                                label={t("assistant.chat.delete")}
                                                onSelect={() => setDeleting(chat)}
                                            />
                                        </Menu>
                                    </li>
                                ))}
                            </ul>
                        </section>
                    ))
                )}
            </div>

            <RenameDialog
                isOpen={renaming !== null}
                title={t("assistant.chat.rename.title")}
                label={t("assistant.chat.rename.label")}
                initialName={renaming?.title ?? ""}
                isPending={rename.isPending}
                errorMessage={rename.isError ? t(aiErrorKey(rename.error)) : null}
                onSubmit={(title) =>
                    renaming &&
                    rename.mutate(
                        { chatId: renaming.id, title },
                        { onSuccess: () => setRenaming(null) },
                    )
                }
                onClose={() => {
                    setRenaming(null);
                    rename.reset();
                }}
            />
            <ConfirmDialog
                isOpen={deleting !== null}
                title={t("assistant.chat.delete.title")}
                message={t("assistant.chat.delete.message")}
                confirmLabel={t("assistant.chat.delete")}
                isPending={remove.isPending}
                isError={remove.isError}
                errorMessage={t(aiErrorKey(remove.error))}
                onConfirm={() =>
                    deleting &&
                    remove.mutate(deleting.id, {
                        onSuccess: () => {
                            if (deleting.id === activeChatId) void navigate("/assistant");
                            setDeleting(null);
                        },
                    })
                }
                onClose={() => {
                    setDeleting(null);
                    remove.reset();
                }}
            />
        </aside>
    );
}
