import { useTranslate } from "../hooks/useTranslate";
import type { Task } from "../types/task";
import { TaskCard } from "./TaskCard";

type TaskListProps = {
  tasks: Task[];
};

export function TaskList({ tasks }: TaskListProps) {
  const t = useTranslate();

  return (
    <section>
      <h2 className="mb-4 text-lg font-semibold">
        {t("workspace.tasks.title")}
      </h2>
      {tasks.length === 0 ? (
        <p className="text-sm text-gray-500">{t("workspace.tasks.empty")}</p>
      ) : (
        <div className="flex flex-col gap-3">
          {tasks.map((task) => (
            <TaskCard key={task.id} task={task} />
          ))}
        </div>
      )}
    </section>
  );
}
