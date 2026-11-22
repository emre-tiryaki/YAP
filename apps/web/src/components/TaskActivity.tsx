import { useState } from "react";
import { FiActivity } from "react-icons/fi";
import { useComments } from "../hooks/useComments";
import { useDateFormat } from "../hooks/useDateFormat";
import { useTaskActivity } from "../hooks/useTaskActivity";
import { useTranslate } from "../hooks/useTranslate";
import type { Comment } from "../types/comment";
import type { TaskEvent } from "../types/taskActivity";
import { isWithinLastMinute } from "../util/date";
import { describeEvent } from "../util/taskEvents";
import { CommentComposer } from "./CommentComposer";
import { CommentItem } from "./CommentItem";
import { InlineError } from "./InlineError";
import { Skeleton } from "./Skeleton";
import { ViewToggle } from "./ViewToggle";

type ActivityFilter = "all" | "comments" | "updates";

type Entry =
    | { kind: "comment"; at: Date; comment: Comment }
    | { kind: "event"; at: Date; event: TaskEvent };

// The task's activity: the change log and the comments in one timeline,
// oldest first, with the comment box at the bottom.
export function TaskActivity({ taskId }: { taskId: string }) {
    const t = useTranslate();
    const format = useDateFormat();
    const comments = useComments(taskId);
    const activity = useTaskActivity(taskId);
    const [filter, setFilter] = useState<ActivityFilter>("all");

    const entries: Entry[] = [
        ...(filter !== "updates" ? comments.data ?? [] : []).map(
            (comment): Entry => ({ kind: "comment", at: comment.createdAt, comment }),
        ),
        ...(filter !== "comments" ? activity.data ?? [] : []).map(
            (event): Entry => ({ kind: "event", at: event.createdAt, event }),
        ),
    ].sort((a, b) => a.at.getTime() - b.at.getTime());

    const isLoading = comments.isLoading || activity.isLoading;
    const isError = comments.isError || activity.isError;

    return (
        <section
            aria-label={t("task.activity.title")}
            className="flex flex-col gap-4"
        >
            <ViewToggle<ActivityFilter>
                value={filter}
                onChange={setFilter}
                label={t("task.activity.filter")}
                options={[
                    { view: "all", label: t("task.activity.all") },
                    { view: "comments", label: t("task.activity.comments") },
                    { view: "updates", label: t("task.activity.updates") },
                ]}
            />
            {isLoading && <Skeleton className="h-16" />}
            {isError && (
                <InlineError
                    message={t("task.detail.commentsLoadError")}
                    onRetry={() => {
                        void comments.refetch();
                        void activity.refetch();
                    }}
                />
            )}
            {!isLoading && !isError && entries.length === 0 && (
                <p className="text-sm text-fg-subtle">
                    {filter === "comments"
                        ? t("task.detail.commentsEmpty")
                        : t("task.activity.empty")}
                </p>
            )}
            {entries.length > 0 && (
                <ol className="flex flex-col gap-3">
                    {entries.map((entry) =>
                        entry.kind === "comment" ? (
                            <CommentItem
                                key={`c-${entry.comment.id}`}
                                taskId={taskId}
                                comment={entry.comment}
                            />
                        ) : (
                            <li
                                key={`e-${entry.event.id}`}
                                className="ml-3 flex items-center gap-2 border-l border-line py-0.5 pl-4 text-caption text-fg-subtle"
                            >
                                <FiActivity aria-hidden className="shrink-0" />
                                <span>
                                    {describeEvent(entry.event, t, format.date)}
                                    {" · "}
                                    <time
                                        dateTime={entry.at.toISOString()}
                                        title={format.dateTime(entry.at)}
                                    >
                                        {format.relative(entry.at) ??
                                            (isWithinLastMinute(entry.at)
                                                ? t("task.detail.justNow")
                                                : format.dateTime(entry.at))}
                                    </time>
                                </span>
                            </li>
                        ),
                    )}
                </ol>
            )}
            <CommentComposer taskId={taskId} />
        </section>
    );
}
