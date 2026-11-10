import type { Project } from "../types/project";
import type { Task } from "../types/task";

const projects: Project[] = [
  {
    id: "0190a1c2-0000-7000-8000-000000000001",
    name: "YAP Backend",
    description: "Go service, SQLite migrations and HTTP handlers.",
    taskCount: 14,
  },
  {
    id: "0190a1c2-0000-7000-8000-000000000002",
    name: "YAP Web",
    description: "React frontend with Tailwind CSS.",
    taskCount: 9,
  },
  {
    id: "0190a1c2-0000-7000-8000-000000000003",
    name: "Capstone Research",
    description: "Literature review and dataset preparation.",
    taskCount: 5,
  },
];

const tasks: Task[] = [
  {
    id: "0190a1c2-0000-7000-8000-000000000101",
    title: "Write workspace repository tests",
    projectId: "0190a1c2-0000-7000-8000-000000000001",
    projectName: "YAP Backend",
    dueDate: new Date("2026-11-04T12:00:00Z"),
    statusCategory: "in_progress",
  },
  {
    id: "0190a1c2-0000-7000-8000-000000000102",
    title: "Build workspace home page",
    projectId: "0190a1c2-0000-7000-8000-000000000002",
    projectName: "YAP Web",
    dueDate: new Date("2026-11-05T12:00:00Z"),
    statusCategory: "in_progress",
  },
  {
    id: "0190a1c2-0000-7000-8000-000000000103",
    title: "Summarize related papers",
    projectId: "0190a1c2-0000-7000-8000-000000000003",
    projectName: "Capstone Research",
    dueDate: new Date("2026-11-09T12:00:00Z"),
    statusCategory: "todo",
  },
];

export function useMockData() {
  return { userName: "Tiryaki", projects, tasks };
}
