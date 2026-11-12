import { useState, type FormEvent, type KeyboardEvent } from "react";
import { FiEdit2, FiMessageSquare, FiTrash2 } from "react-icons/fi";
import {
    useComments,
    useCreateComment,
    useDeleteComment,
    useUpdateComment,
} from "../hooks/useComments";
import { useTranslate } from "../hooks/useTranslate";
import type { Comment } from "../types/comment";
import { formatRelativeTime } from "../util/date";
import { FormError } from "./FormError";
import { InlineError } from "./InlineError";

type CommentSectionProps = {
    taskId: string;
};

// Ctrl/Cmd + Enter sends the textarea's form; plain Enter adds a new line.
function submitOnModEnter(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
        event.preventDefault();
        event.currentTarget.form?.requestSubmit();
    }
}

export function CommentSection({ taskId }: CommentSectionProps) {
    const t = useTranslate();
    const comments = useComments(taskId);
    const create = useCreateComment(taskId);
    const [draft, setDraft] = useState("");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        const body = draft.trim();
        if (body === "") return;
        create.mutate(body, { onSuccess: () => setDraft("") });
    }

    const items = comments.data ?? [];

    return (
        <section
            aria-labelledby="task-comments-heading"
            className="flex flex-col gap-4"
        >
            <div className="flex items-center gap-3">
                <FiMessageSquare aria-hidden className="text-gray-500" />
                <h3
                    id="task-comments-heading"
                    className="text-sm font-semibold text-gray-800"
                >
                    {t("task.detail.comments")}
                </h3>
                {items.length > 0 && (
                    <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
                        {items.length}
                    </span>
                )}
            </div>

            {comments.isLoading && (
                <div
                    aria-busy="true"
                    className="h-16 animate-pulse rounded-md bg-gray-100"
                />
            )}
            {comments.isError && (
                <InlineError
                    message={t("task.detail.commentsLoadError")}
                    onRetry={() => void comments.refetch()}
                />
            )}
            {comments.isSuccess && items.length === 0 && (
                <p className="text-sm text-gray-500">
                    {t("task.detail.commentsEmpty")}
                </p>
            )}
            {items.length > 0 && (
                <ol className="flex flex-col gap-4">
                    {items.map((comment) => (
                        <CommentItem
                            key={comment.id}
                            taskId={taskId}
                            comment={comment}
                        />
                    ))}
                </ol>
            )}

            <form
                onSubmit={handleSubmit}
                className="flex flex-col gap-2 rounded-lg border border-gray-300 bg-white p-2 focus-within:border-gray-500"
            >
                <textarea
                    aria-label={t("task.detail.commentPlaceholder")}
                    placeholder={t("task.detail.commentPlaceholder")}
                    value={draft}
                    rows={2}
                    disabled={create.isPending}
                    onChange={(event) => setDraft(event.target.value)}
                    onKeyDown={submitOnModEnter}
                    className="field-sizing-content max-h-60 min-h-12 w-full resize-none px-1 text-sm focus:outline-none"
                />
                <div className="flex items-center justify-between gap-2">
                    <span className="text-xs text-gray-400">
                        {t("task.detail.commentHint")}
                    </span>
                    <button
                        type="submit"
                        disabled={create.isPending || draft.trim() === ""}
                        className="cursor-pointer rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                        {t("task.detail.commentSubmit")}
                    </button>
                </div>
            </form>
            <FormError
                message={create.isError ? t("task.detail.commentError") : null}
            />
        </section>
    );
}

type CommentItemProps = {
    taskId: string;
    comment: Comment;
};

function CommentItem({ taskId, comment }: CommentItemProps) {
    const t = useTranslate();
    const update = useUpdateComment(taskId);
    const remove = useDeleteComment(taskId);
    const [isEditing, setIsEditing] = useState(false);
    const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
    const [draft, setDraft] = useState(comment.body);

    const when =
        formatRelativeTime(comment.createdAt) ?? t("task.detail.justNow");
    const isEdited = comment.updatedAt.getTime() > comment.createdAt.getTime();

    function cancelEdit() {
        setDraft(comment.body);
        setIsEditing(false);
        update.reset();
    }

    function handleSave(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        const body = draft.trim();
        if (body === "") return;
        if (body === comment.body) {
            setIsEditing(false);
            return;
        }
        update.mutate(
            { commentId: comment.id, body },
            { onSuccess: () => setIsEditing(false) },
        );
    }

    return (
        <li className="group flex gap-3">
            <span
                aria-hidden
                className="flex size-8 shrink-0 items-center justify-center rounded-full bg-gray-800 text-xs font-semibold text-white uppercase"
            >
                {comment.authorName.trim().charAt(0) || "?"}
            </span>
            <div className="flex min-w-0 flex-1 flex-col gap-1">
                <div className="flex items-baseline gap-2">
                    <span className="text-sm font-medium text-gray-900">
                        {comment.authorName}
                    </span>
                    <time
                        dateTime={comment.createdAt.toISOString()}
                        title={comment.createdAt.toLocaleString()}
                        className="text-xs text-gray-500"
                    >
                        {when}
                    </time>
                    {isEdited && (
                        <span
                            title={comment.updatedAt.toLocaleString()}
                            className="text-xs text-gray-400"
                        >
                            ({t("task.detail.commentEdited")})
                        </span>
                    )}
                    {comment.isOwn && !isEditing && (
                        <div className="ml-auto flex items-center gap-1 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
                            <button
                                type="button"
                                onClick={() => setIsEditing(true)}
                                aria-label={t("task.detail.editComment")}
                                title={t("task.detail.editComment")}
                                className="cursor-pointer rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
                            >
                                <FiEdit2 aria-hidden className="size-3.5" />
                            </button>
                            <button
                                type="button"
                                onClick={() => setIsConfirmingDelete(true)}
                                aria-label={t("task.detail.deleteComment")}
                                title={t("task.detail.deleteComment")}
                                className="cursor-pointer rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
                            >
                                <FiTrash2 aria-hidden className="size-3.5" />
                            </button>
                        </div>
                    )}
                </div>

                {isEditing ? (
                    <form onSubmit={handleSave} className="flex flex-col gap-2">
                        <textarea
                            aria-label={t("task.detail.editComment")}
                            value={draft}
                            autoFocus
                            disabled={update.isPending}
                            onChange={(event) => setDraft(event.target.value)}
                            onKeyDown={(event) => {
                                if (event.key === "Escape") {
                                    // Cancels the edit, not the whole dialog.
                                    event.stopPropagation();
                                    cancelEdit();
                                    return;
                                }
                                submitOnModEnter(event);
                            }}
                            className="field-sizing-content min-h-16 w-full resize-none rounded-md border border-gray-300 p-2 text-sm focus:border-gray-500 focus:outline-none"
                        />
                        <div className="flex gap-2">
                            <button
                                type="submit"
                                disabled={update.isPending || draft.trim() === ""}
                                className="cursor-pointer rounded-md bg-gray-900 px-3 py-1 text-sm text-white hover:bg-gray-700 disabled:cursor-not-allowed disabled:opacity-50"
                            >
                                {t("common.save")}
                            </button>
                            <button
                                type="button"
                                onClick={cancelEdit}
                                className="cursor-pointer rounded-md px-3 py-1 text-sm text-gray-600 hover:bg-gray-100"
                            >
                                {t("common.cancel")}
                            </button>
                        </div>
                        <FormError
                            message={
                                update.isError
                                    ? t("task.detail.commentError")
                                    : null
                            }
                        />
                    </form>
                ) : (
                    <p className="rounded-lg rounded-tl-none bg-gray-50 px-3 py-2 text-sm wrap-break-word whitespace-pre-wrap text-gray-800">
                        {comment.body}
                    </p>
                )}

                {isConfirmingDelete && (
                    <div
                        role="alert"
                        className="flex flex-wrap items-center gap-2 text-sm"
                    >
                        <span className="text-gray-700">
                            {t("task.detail.deleteCommentConfirm")}
                        </span>
                        <button
                            type="button"
                            disabled={remove.isPending}
                            onClick={() => remove.mutate(comment.id)}
                            className="cursor-pointer rounded-md bg-red-600 px-2.5 py-1 text-xs text-white hover:bg-red-700 disabled:opacity-50"
                        >
                            {t("common.delete")}
                        </button>
                        <button
                            type="button"
                            onClick={() => {
                                setIsConfirmingDelete(false);
                                remove.reset();
                            }}
                            className="cursor-pointer rounded-md px-2.5 py-1 text-xs text-gray-600 hover:bg-gray-100"
                        >
                            {t("common.cancel")}
                        </button>
                        {remove.isError && (
                            <span className="text-xs text-red-600">
                                {t("common.error")}
                            </span>
                        )}
                    </div>
                )}
            </div>
        </li>
    );
}
