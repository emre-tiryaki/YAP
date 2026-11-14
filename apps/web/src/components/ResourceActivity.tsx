import type { ReactNode } from "react";
import { Link } from "react-router";
import { useDateFormat } from "../hooks/useDateFormat";
import { useResourceEvents } from "../hooks/useResources";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { ResourceEvent, ResourceEventScope } from "../types/resource";
import { isWithinLastMinute } from "../util/date";
import { formatSize } from "../util/resources";

type ResourceActivityProps = {
    scope: ResourceEventScope;
    emptyText?: string;
};

// The activity log: who did what to which resource, newest first.
export function ResourceActivity({ scope, emptyText }: ResourceActivityProps) {
    const t = useTranslate();
    const events = useResourceEvents(scope);
    const items = events.data?.pages.flat() ?? [];

    if (events.isLoading) return <p className="text-sm text-gray-400">{t("common.loading")}</p>;
    if (events.isError)
        return (
            <p role="alert" className="text-sm text-red-600">
                {t("common.error")}
            </p>
        );
    if (items.length === 0)
        return <p className="text-sm text-gray-400">{emptyText ?? t("resources.activity.empty")}</p>;

    return (
        <div className="flex flex-col gap-3">
            <ol className="flex flex-col gap-3">
                {items.map((event) => (
                    <EventItem key={event.id} event={event} />
                ))}
            </ol>
            {events.hasNextPage && (
                <button
                    type="button"
                    onClick={() => void events.fetchNextPage()}
                    disabled={events.isFetchingNextPage}
                    className="cursor-pointer self-start rounded-md border border-gray-300 px-3 py-1 text-xs text-gray-600 hover:bg-gray-100 disabled:opacity-50"
                >
                    {t(events.isFetchingNextPage ? "common.loading" : "resources.activity.more")}
                </button>
            )}
        </div>
    );
}

// Fills a template like "{actor} renamed {from} to {name}", turning the
// placeholders into the given nodes.
function fill(template: string, values: Record<string, ReactNode>): ReactNode[] {
    return template.split(/(\{\w+\})/).map((part, i) => {
        const match = /^\{(\w+)\}$/.exec(part);
        return match && match[1] in values ? <span key={i}>{values[match[1]]}</span> : part;
    });
}

function EventItem({ event }: { event: ResourceEvent }) {
    const t = useTranslate();
    const format = useDateFormat();
    const d = event.details;
    const top = t("resources.title");
    const key = `resources.event.${event.action}${event.details.replaced ? ".replaced" : ""}` as TranslationKey;
    const merged = event.updatedAt.getTime() - event.createdAt.getTime() > 60_000;

    return (
        <li className="flex gap-3 text-sm">
            <span
                aria-hidden
                className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-gray-200 text-xs font-semibold text-gray-700"
            >
                {event.actorName.charAt(0).toUpperCase()}
            </span>
            <div className="min-w-0 flex-1">
                <p className="break-words text-gray-700">
                    {fill(t(key), {
                        actor: <strong className="font-medium text-gray-900">{event.actorName}</strong>,
                        name: (
                            <Link
                                to={`/resources/${event.resourceId}`}
                                className="font-medium text-gray-900 hover:underline"
                            >
                                {d.name}
                            </Link>
                        ),
                        from: <em className="not-italic text-gray-900">{d.from || top}</em>,
                        to: <em className="not-italic text-gray-900">{d.to || top}</em>,
                        task: <em className="not-italic text-gray-900">{d.taskTitle}</em>,
                    })}
                </p>
                <p className="mt-0.5 text-xs text-gray-400">
                    <time dateTime={event.createdAt.toISOString()} title={format.dateTime(event.createdAt)}>
                        {format.relative(event.createdAt) ??
                            (isWithinLastMinute(event.createdAt)
                                ? t("task.detail.justNow")
                                : format.dateTime(event.createdAt))}
                    </time>
                    {merged && (
                        <>
                            {" – "}
                            <time dateTime={event.updatedAt.toISOString()}>{format.time(event.updatedAt)}</time>
                        </>
                    )}
                    {d.size !== undefined && event.action !== "linked" && ` · ${formatSize(d.size)}`}
                </p>
            </div>
        </li>
    );
}
