import { useTranslate } from "../hooks/useTranslate";
import type { Task } from "../types/task";

type TaskCardProps = {
  task: Task;
};

export function TaskCard({ task }: TaskCardProps) {
  const t = useTranslate();

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <h3 className="font-semibold">{task.title}</h3>
      <p className="mt-1 text-sm text-gray-600">{task.projectName}</p>
      <p className="mt-3 text-xs text-gray-500">
        {t("workspace.tasks.dueDate")}:{" "}
        {new Date(task.dueDate).toLocaleDateString()}
      </p>
    </div>
  );
}
