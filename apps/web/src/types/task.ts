export type StatusCategory = "todo" | "in_progress" | "done";

export type Task = {
  id: string;
  title: string;
  projectName: string;
  dueDate: Date | null;
  statusCategory: StatusCategory;
};

export type CreateTaskInput = {
  title: string
  dueDate: Date | null
}