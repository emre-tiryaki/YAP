export type StatusCategory = "todo" | "in_progress" | "done";

// Properties a task row or card can show inline; the display options (T13)
// choose which.
export type TaskPropertyKey =
  | "status"
  | "priority"
  | "dueDate"
  | "subtasks"
  | "comments";

export type TaskPriority = "urgent" | "high" | "medium" | "low" | "none";

export type PlanTask = {
  id: string;
  title: string;
  description: string;
  statusId: string;
  statusCategory: StatusCategory;
  position: string;
  dueDate: Date | null;
  priority: TaskPriority;
  // The task's number in its project (the 12 of ABC-12); 0 while it is
  // still being created.
  number: number;
  // Set while the task is finished; independent of the status it sits in.
  completedAt: Date | null;
  subtaskCount: number;
  doneSubtaskCount: number;
  commentCount: number;
};

// A task outside its plan page (e.g. the upcoming list) also names where it lives.
export type Task = PlanTask & {
  projectId: string;
  projectName: string;
  planId: string;
};

export type CreateTaskInput = {
  title: string
  dueDate: Date | null
  priority?: TaskPriority
  description?: string
  planId?: string
  statusId?: string
}

export type MoveTaskInput = {
  statusId: string
  // Position among the other tasks of the target status.
  index: number
}

// Omitted fields are kept; a null due date clears it.
export type UpdateTaskInput = {
  title?: string
  description?: string
  dueDate?: Date | null
  priority?: TaskPriority
}

// A direct child of a task, shown as a checklist item inside its parent.
export type Subtask = {
  id: string
  parentId: string
  title: string
  planId: string
  statusId: string
  statusCategory: StatusCategory
  completedAt: Date | null
}
