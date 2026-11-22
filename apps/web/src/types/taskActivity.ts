export type TaskEventAction =
    | "created"
    | "renamed"
    | "described"
    | "status_changed"
    | "priority_changed"
    | "due_changed"
    | "start_changed"
    | "labels_changed"
    | "done"
    | "reopened"
    | "relation_added"
    | "relation_removed";

// One entry of a task's activity log. `details` holds strings: from and to
// for a changed field, kind (a RelationView) and task (the other task's
// title) for a relation.
export type TaskEvent = {
    id: string;
    action: TaskEventAction;
    details: Record<string, string>;
    createdAt: Date;
};

// As stored: the first task blocks, relates to or duplicates the second.
export type RelationKind = "blocks" | "relates" | "duplicates";

// How a task reads one of its relations.
export type RelationView =
    | "blocking"
    | "blocked_by"
    | "relates"
    | "duplicate_of"
    | "duplicated_by";

export type TaskRelation = {
    view: RelationView;
    // The other task.
    taskId: string;
    title: string;
    number: number;
    projectId: string;
    planId: string;
    completedAt: Date | null;
};
