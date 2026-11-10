import { useState } from "react";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Task } from "../types/task";
import { CardMenu } from "./CardMenu";
import { TaskDialogs } from "./TaskDialogs";

type TaskCardProps = {
  task: Task;
};

export function TaskCard({ task }: TaskCardProps) {
  const t = useTranslate();
  const [action, setAction] = useState<CardAction | null>(null);

  return (
    <div className="relative rounded-lg border border-gray-200 bg-white p-4 pr-11 shadow-sm">
      <h3 className="font-semibold">{task.title}</h3>
      <p className="mt-1 text-sm text-gray-600">{task.projectName}</p>
      {task.dueDate && (
        <p className="mt-3 text-xs text-gray-500">
          {t('workspace.tasks.dueDate')}: {task.dueDate.toLocaleDateString()}
        </p>
      )}
      <div className="absolute top-3 right-3">
        <CardMenu
          onSelect={setAction}
          pin={{
            kind: "task",
            id: task.id,
            label: task.title,
            href: `/projects/${task.projectId}`,
            projectId: task.projectId,
          }}
        />
      </div>
      {/* The upcoming list carries no description, so the edit dialog hides that field. */}
      <TaskDialogs task={task} action={action} onClose={() => setAction(null)} />
    </div>
  );
}
