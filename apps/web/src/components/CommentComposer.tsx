import { useState, type FormEvent } from "react";
import { useCreateComment } from "../hooks/useComments";
import { useTranslate } from "../hooks/useTranslate";
import { submitOnModEnter } from "../util/submitOnModEnter";
import { Button } from "./Button";
import { FormError } from "./FormError";

// The box under the activity timeline to write a new comment.
export function CommentComposer({ taskId }: { taskId: string }) {
    const t = useTranslate();
    const create = useCreateComment(taskId);
    const [draft, setDraft] = useState("");

    function handleSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        const body = draft.trim();
        if (body === "") return;
        create.mutate(body, { onSuccess: () => setDraft("") });
    }

    return (
        <>
            <form
                onSubmit={handleSubmit}
                className="flex flex-col gap-2 rounded-lg border border-line-strong bg-surface-1 p-2 focus-within:border-gray-500"
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
                    <span className="text-xs text-fg-placeholder">
                        {t("task.detail.commentHint")}
                    </span>
                    <Button variant="primary" size="lg"
                        type="submit"
                        disabled={create.isPending || draft.trim() === ""}
                    >
                        {t("task.detail.commentSubmit")}
                    </Button>
                </div>
            </form>
            <FormError
                message={create.isError ? t("task.detail.commentError") : null}
            />
        </>
    );
}
