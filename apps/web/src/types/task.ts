export type StatusCategory = "todo" | "in_progress" | "done";

export type Task = {
  id: string;
  title: string;
  projectId: string;
  projectName: string;
  dueDate: Date | null;
  statusCategory: StatusCategory;
};

export type PlanTask = {
  id: string;
  title: string;
  description: string;
  statusId: string;
  statusCategory: StatusCategory;
  position: string;
  dueDate: Date | null;
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
