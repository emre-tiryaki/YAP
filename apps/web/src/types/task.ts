export type StatusCategory = "todo" | "in_progress" | "done";

export type PlanTask = {
  id: string;
  title: string;
  description: string;
  statusId: string;
  statusCategory: StatusCategory;
  position: string;
  dueDate: Date | null;
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
