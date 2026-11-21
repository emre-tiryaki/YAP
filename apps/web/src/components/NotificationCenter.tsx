import { useEffect, useState, type ComponentType } from "react";
import { useNavigate } from "react-router";
import { FiAlertCircle, FiCheck, FiClock, FiMessageSquare, FiX } from "react-icons/fi";
import { useRecentComments } from "../hooks/useComments";
import { useDateFormat } from "../hooks/useDateFormat";
import { useUpcomingTasks } from "../hooks/useTasks";
import { dismissToast, pushToasts, useToasts } from "../hooks/useToasts";
import { useTranslate } from "../hooks/useTranslate";
import { useUserSettings } from "../hooks/useUserSettings";
import type {
    AppNotification,
    AppNotificationKind,
} from "../types/notification";
import { playNotificationSound } from "../util/notificationSound";
import {
    commentNotification,
    readCommentsSince,
    takeUnsent,
    taskNotifications,
    writeCommentsSince,
} from "../util/notifications";

// How often due dates are checked and new comments fetched.
const pollMs = 60_000;
const toastMs = 10_000;

type IconType = ComponentType<{ className?: string; "aria-hidden"?: boolean }>;

const icons: Record<AppNotificationKind, { Icon: IconType; tone: string }> = {
    dueSoon: { Icon: FiClock, tone: "text-accent" },
    overdue: { Icon: FiAlertCircle, tone: "text-red-600" },
    comment: { Icon: FiMessageSquare, tone: "text-gray-500" },
    info: { Icon: FiCheck, tone: "text-accent" },
};

// Watches the user's tasks and new comments and shows the notifications the
// settings ask for: in the app, plus as system notifications while the tab
// is in the background (when the browser allows them).
export function NotificationCenter() {
    const t = useTranslate();
    const { locale } = useDateFormat();
    const navigate = useNavigate();
    const settings = useUserSettings();
    // Nothing is shown before the stored settings are known.
    const notify = settings.data?.notifications;
    const wantsTasks = notify !== undefined && (notify.dueSoon || notify.overdue);
    const wantsComments = notify?.comments ?? false;
    const tasks = useUpcomingTasks(wantsTasks ? pollMs : undefined);
    const comments = useRecentComments(wantsComments, pollMs);
    // Re-checks due dates as time passes, also without new data.
    const [tick, setTick] = useState(0);

    useEffect(() => {
        const timer = setInterval(() => setTick((n) => n + 1), pollMs);
        return () => clearInterval(timer);
    }, []);

    // Turning comment notifications off forgets where they were, so turning
    // them back on does not bring up everything written in between.
    useEffect(() => {
        if (settings.data && !wantsComments) writeCommentsSince(null);
    }, [settings.data, wantsComments]);

    useEffect(() => {
        if (!notify) return;
        const now = new Date();
        const items: AppNotification[] = [];
        if (tasks.data)
            items.push(...taskNotifications(tasks.data, notify, now, t, locale));
        if (notify.comments && comments.data && comments.data.length > 0) {
            items.push(...comments.data.map((c) => commentNotification(c, t)));
            const latest = comments.data[comments.data.length - 1].createdAt;
            const since = readCommentsSince();
            if (!since || latest > since) writeCommentsSince(latest);
        }
        const fresh = takeUnsent(items, now);
        if (fresh.length === 0) return;

        pushToasts(fresh);
        if (
            document.hidden &&
            "Notification" in window &&
            Notification.permission === "granted"
        ) {
            for (const item of fresh) {
                const system = new Notification(item.title, {
                    body: item.body,
                    tag: item.id,
                });
                system.onclick = () => {
                    window.focus();
                    if (item.href) void navigate(item.href);
                    dismissToast(item.id);
                    system.close();
                };
            }
        }
        if (notify.sound) playNotificationSound();
    }, [notify, tasks.data, comments.data, tick, t, locale, navigate]);

    return <Toasts />;
}

function Toasts() {
    const t = useTranslate();
    const toasts = useToasts();
    if (toasts.length === 0) return null;
    return (
        <section
            aria-label={t("notifications.label")}
            aria-live="polite"
            className="fixed right-4 bottom-4 z-40 flex w-80 flex-col gap-2"
        >
            {toasts.map((toast) => (
                <Toast key={toast.id} toast={toast} />
            ))}
        </section>
    );
}

function Toast({ toast }: { toast: AppNotification }) {
    const t = useTranslate();
    const navigate = useNavigate();
    const { Icon, tone } = icons[toast.kind];

    useEffect(() => {
        const timer = setTimeout(() => dismissToast(toast.id), toastMs);
        return () => clearTimeout(timer);
    }, [toast.id]);

    const content = (
        <>
            <Icon aria-hidden className={`mt-0.5 shrink-0 ${tone}`} />
            <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium text-gray-900">
                    {toast.title}
                </span>
                {toast.body && (
                    <span className="line-clamp-2 block text-xs text-gray-500">
                        {toast.body}
                    </span>
                )}
            </span>
        </>
    );

    return (
        <div className="flex items-start gap-1 rounded-lg border border-gray-200 bg-white p-3 shadow-lg">
            {toast.href ? (
                <button
                    type="button"
                    onClick={() => {
                        if (toast.href) void navigate(toast.href);
                        dismissToast(toast.id);
                    }}
                    className="flex min-w-0 flex-1 cursor-pointer items-start gap-3 text-left"
                >
                    {content}
                </button>
            ) : (
                <div className="flex min-w-0 flex-1 items-start gap-3">
                    {content}
                </div>
            )}
            <button
                type="button"
                onClick={() => dismissToast(toast.id)}
                aria-label={t("common.close")}
                title={t("common.close")}
                className="shrink-0 cursor-pointer rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
            >
                <FiX aria-hidden />
            </button>
        </div>
    );
}
