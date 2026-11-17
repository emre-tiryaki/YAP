import type { TranslationKey } from "../hooks/useTranslate";
import type { AppNotification, RecentComment } from "../types/notification";
import type { Task } from "../types/task";
import type { NotificationSettings, Reminder } from "../types/userSettings";
import { daysFromToday, toDateInputValue } from "./date";

// Which notifications were already shown on this device, by id, with the time
// they were shown. Kept per device: each device reminds on its own.
const SENT_KEY = "yap.notifications.sent";
// Comments written from this time on are notified; set when the comment
// notifications are turned on.
const SINCE_KEY = "yap.notifications.since";
// Entries older than this are dropped; no notification id repeats after it.
const SENT_TTL_MS = 30 * 86_400_000;

const reminderMs: Record<Reminder, number> = {
    "1h": 3_600_000,
    "1d": 86_400_000,
    "1w": 7 * 86_400_000,
};

type Translate = (key: TranslationKey) => string;

export function taskHref(task: {
    projectId: string;
    planId: string;
    id: string;
}): string {
    return `/projects/${task.projectId}/plans/${task.planId}?task=${task.id}`;
}

// A due date is a local calendar day (stored as its local midnight). The
// reminder counts back from the start of that day; the task is overdue once
// the day is over.
function dayStart(due: Date): number {
    return new Date(due.getFullYear(), due.getMonth(), due.getDate()).getTime();
}

function dayEnd(due: Date): number {
    return new Date(
        due.getFullYear(),
        due.getMonth(),
        due.getDate() + 1,
    ).getTime();
}

export function isOverdue(due: Date, now: Date): boolean {
    return now.getTime() >= dayEnd(due);
}

export function isDueSoon(due: Date, reminder: Reminder, now: Date): boolean {
    const t = now.getTime();
    return t >= dayStart(due) - reminderMs[reminder] && t < dayEnd(due);
}

function dueLabel(due: Date, now: Date, t: Translate, locale: string): string {
    const days = daysFromToday(due, now);
    if (days === 0) return t("notifications.dueToday");
    if (days === 1) return t("notifications.dueTomorrow");
    return t("notifications.dueOn").replace(
        "{date}",
        due.toLocaleDateString(locale, { weekday: "long", day: "numeric", month: "long" }),
    );
}

// The notifications the user's open tasks call for right now. Overdue tasks
// are summed up in one notification a day.
export function taskNotifications(
    tasks: Task[],
    settings: NotificationSettings,
    now: Date,
    t: Translate,
    locale: string,
): AppNotification[] {
    const out: AppNotification[] = [];
    const overdue: Task[] = [];
    for (const task of tasks) {
        if (!task.dueDate || task.completedAt) continue;
        if (isOverdue(task.dueDate, now)) {
            overdue.push(task);
            continue;
        }
        if (settings.dueSoon && isDueSoon(task.dueDate, settings.reminder, now)) {
            out.push({
                id: `due:${task.id}:${toDateInputValue(task.dueDate)}:${settings.reminder}`,
                kind: "dueSoon",
                title: task.title,
                body: `${dueLabel(task.dueDate, now, t, locale)} · ${task.projectName}`,
                href: taskHref(task),
            });
        }
    }
    if (settings.overdue && overdue.length > 0) {
        const [first] = overdue;
        out.push({
            id: `overdue:${toDateInputValue(now)}`,
            kind: "overdue",
            title: t("notifications.overdue.title").replace(
                "{count}",
                String(overdue.length),
            ),
            body: overdue
                .slice(0, 3)
                .map((task) => task.title)
                .join(", ")
                .concat(overdue.length > 3 ? ", …" : ""),
            href: overdue.length === 1 ? taskHref(first) : "/home",
        });
    }
    return out;
}

const maxCommentPreview = 140;

export function commentNotification(
    comment: RecentComment,
    t: Translate,
): AppNotification {
    const body = comment.body.replace(/\s+/g, " ").trim();
    return {
        id: `comment:${comment.id}`,
        kind: "comment",
        title: t("notifications.comment.title")
            .replace("{author}", comment.authorName)
            .replace("{task}", comment.taskTitle),
        body:
            body.length > maxCommentPreview
                ? `${body.slice(0, maxCommentPreview)}…`
                : body,
        href: taskHref({
            projectId: comment.projectId,
            planId: comment.planId,
            id: comment.parentId ?? comment.taskId,
        }),
    };
}

function readSent(): Record<string, number> {
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(SENT_KEY) ?? "{}");
        if (typeof parsed === "object" && parsed !== null)
            return parsed as Record<string, number>;
    } catch {
        // Corrupt value: start over.
    }
    return {};
}

// Returns the notifications not shown before and records them as shown.
export function takeUnsent(
    notifications: AppNotification[],
    now: Date = new Date(),
): AppNotification[] {
    if (notifications.length === 0) return [];
    const sent = readSent();
    const fresh = notifications.filter((n) => !(n.id in sent));
    if (fresh.length === 0) return [];
    const cutoff = now.getTime() - SENT_TTL_MS;
    const next: Record<string, number> = {};
    for (const [id, at] of Object.entries(sent)) {
        if (typeof at === "number" && at >= cutoff) next[id] = at;
    }
    for (const n of fresh) next[n.id] = now.getTime();
    localStorage.setItem(SENT_KEY, JSON.stringify(next));
    return fresh;
}

// The time comment notifications start from, or null while they are off.
export function readCommentsSince(): Date | null {
    const raw = localStorage.getItem(SINCE_KEY);
    if (!raw) return null;
    const date = new Date(raw);
    return Number.isNaN(date.getTime()) ? null : date;
}

export function writeCommentsSince(date: Date | null) {
    if (date) localStorage.setItem(SINCE_KEY, date.toISOString());
    else localStorage.removeItem(SINCE_KEY);
}
