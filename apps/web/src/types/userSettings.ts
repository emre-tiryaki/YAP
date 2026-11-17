import type { StatusCategory } from "./task";

// How long before a task's due date the reminder is shown.
export type Reminder = "1h" | "1d" | "1w";

export type NotificationSettings = {
    dueSoon: boolean;
    reminder: Reminder;
    overdue: boolean;
    comments: boolean;
    sound: boolean;
};

// Settings stored in the database, so they follow the user to every device.
export type UserSettings = {
    autosaveEnabled: boolean;
    autosaveDelayMs: number;
    // Completed tasks are left out of plan boards and lists.
    hideDone: boolean;
    // Completed tasks come after the open ones of each status.
    doneToBottom: boolean;
    // Titles of the statuses a new plan starts with.
    statusNames: Record<StatusCategory, string>;
    notifications: NotificationSettings;
};

export type UpdateUserSettingsInput = Partial<
    Omit<UserSettings, "statusNames" | "notifications">
> & {
    statusNames?: Partial<Record<StatusCategory, string>>;
    notifications?: Partial<NotificationSettings>;
};
