// A comment another member wrote, as shown in a notification.
export type RecentComment = {
    id: string;
    taskId: string;
    taskTitle: string;
    projectId: string;
    planId: string;
    // The top-level task when the comment is on a subtask; null otherwise.
    parentId: string | null;
    authorName: string;
    body: string;
    createdAt: Date;
};

export type AppNotificationKind = "dueSoon" | "overdue" | "comment";

// A notification shown in the app (and as a system notification when the
// browser allows it).
export type AppNotification = {
    // Also the key that keeps it from being shown twice.
    id: string;
    kind: AppNotificationKind;
    title: string;
    body: string;
    // Where clicking it leads; null when it has no single target.
    href: string | null;
};
