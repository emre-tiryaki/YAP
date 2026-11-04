import { useTranslate } from "../hooks/useTranslate";
import type { Task } from "../types/task";
import { TaskCard } from "./TaskCard";

type TaskListProps = {
    tasks: Task[];
    isLoading: boolean;
    isError: boolean;
};

export function TaskList({ tasks, isError, isLoading }: TaskListProps) {
    const t = useTranslate();

    return (
        <section>
            <h2 className="mb-4 text-lg font-semibold">
                {t("workspace.tasks.title")}
            </h2>
            {isLoading && (
                <p className="text-sm text-gray-500">{t("common.loading")}</p>
            )}
            {isError && (
                <p className="text-sm text-red-600">{t("common.error")}</p>
            )}
            {!isLoading && !isError && tasks.length === 0 && (
                <p className="text-sm text-gray-500">
                    {t("workspace.tasks.empty")}
                </p>
            )}
            {tasks.length > 0 && (
                <div className="flex flex-col gap-3">
                    {tasks.map((task) => (
                        <TaskCard key={task.id} task={task} />
                    ))}
                </div>
            )}
        </section>
    );
}
