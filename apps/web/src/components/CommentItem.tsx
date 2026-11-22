import { useState, type FormEvent } from "react";
import { FiEdit2, FiTrash2 } from "react-icons/fi";
import { useDeleteComment, useUpdateComment } from "../hooks/useComments";
import { useTranslate } from "../hooks/useTranslate";
import type { Comment } from "../types/comment";
import { useDateFormat } from "../hooks/useDateFormat";
import { isWithinLastMinute } from "../util/date";
import { submitOnModEnter } from "../util/submitOnModEnter";
import { ExpandableText } from "./ExpandableText";
import { FormError } from "./FormError";
import { Button } from "./Button";

type CommentItemProps = {
    taskId: string;
    comment: Comment;
};

export function CommentItem({ taskId, comment }: CommentItemProps) {
    const t = useTranslate();
    const update = useUpdateComment(taskId);
    const remove = useDeleteComment(taskId);
    const [isEditing, setIsEditing] = useState(false);
    const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
    const [draft, setDraft] = useState(comment.body);

    const format = useDateFormat();
    const when =
        format.relative(comment.createdAt) ??
        (isWithinLastMinute(comment.createdAt)
            ? t("task.detail.justNow")
            : format.dateTime(comment.createdAt));
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
                    <span className="text-sm font-medium text-fg">
                        {comment.authorName}
                    </span>
                    <time
                        dateTime={comment.createdAt.toISOString()}
                        title={format.dateTime(comment.createdAt)}
                        className="text-xs text-fg-subtle"
                    >
                        {when}
                    </time>
                    {isEdited && (
                        <span
                            title={format.dateTime(comment.updatedAt)}
                            className="text-xs text-fg-placeholder"
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
                                className="cursor-pointer rounded p-1 text-fg-placeholder hover:bg-layer-1-hover hover:text-fg-muted"
                            >
                                <FiEdit2 aria-hidden className="size-3.5" />
                            </button>
                            <button
                                type="button"
                                onClick={() => setIsConfirmingDelete(true)}
                                aria-label={t("task.detail.deleteComment")}
                                title={t("task.detail.deleteComment")}
                                className="cursor-pointer rounded p-1 text-fg-placeholder hover:bg-red-50 hover:text-red-600"
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
                            className="field-sizing-content min-h-16 w-full resize-none rounded-md border border-line-strong p-2 text-sm focus:border-gray-500 focus:outline-none"
                        />
                        <div className="flex gap-2">
                            <Button variant="primary" size="md"
                                type="submit"
                                disabled={update.isPending || draft.trim() === ""}
                            >
                                {t("common.save")}
                            </Button>
                            <button
                                type="button"
                                onClick={cancelEdit}
                                className="cursor-pointer rounded-md px-3 py-1 text-sm text-fg-muted hover:bg-layer-1-hover"
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
                    <div className="rounded-lg rounded-tl-none bg-layer-1 px-3 py-2">
                        <ExpandableText
                            text={comment.body}
                            className="text-sm text-fg"
                        />
                    </div>
                )}

                {isConfirmingDelete && (
                    <div
                        role="alert"
                        className="flex flex-wrap items-center gap-2 text-sm"
                    >
                        <span className="text-fg-muted">
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
                            className="cursor-pointer rounded-md px-2.5 py-1 text-xs text-fg-muted hover:bg-layer-1-hover"
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
