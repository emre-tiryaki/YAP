export type StatusCategory = "todo" | "in_progress" | "done";

export type Task = {
  id: string;
  title: string;
  projectName: string;
  dueDate: string;
  statusCategory: StatusCategory;
};
