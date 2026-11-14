import { useState } from "react";
import { useProjectPlans, useProjects } from "../hooks/useProjects";
import { usePlanTasks } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import { Modal } from "./Modal";
import { SelectInput } from "./SelectInput";

type TaskPickerDialogProps = {
    title: string;
    isPending?: boolean;
    errorMessage?: string | null;
    onPick: (taskId: string) => void;
    onClose: () => void;
};

// Picks a task by project, plan and title, to link a resource to it.
export function TaskPickerDialog({
    title,
    isPending = false,
    errorMessage,
    onPick,
    onClose,
}: TaskPickerDialogProps) {
    const t = useTranslate();
    const projects = useProjects();
    const [projectId, setProjectId] = useState("");
    const plans = useProjectPlans(projectId);
    const [planId, setPlanId] = useState("");
    const tasks = usePlanTasks(projectId, planId);
    const [taskId, setTaskId] = useState("");

    const choose = (label: string) => ({ value: "", label: `— ${label} —` });

    return (
        <Modal isOpen title={title} onClose={onClose}>
            <div className="flex flex-col gap-3">
                <SelectInput
                    label={t("resources.link.project")}
                    value={projectId}
                    onChange={(event) => {
                        setProjectId(event.target.value);
                        setPlanId("");
                        setTaskId("");
                    }}
                    options={[
                        choose(t("resources.link.project")),
                        ...(projects.data ?? []).map((p) => ({
                            value: p.id,
                            label: p.icon ? `${p.icon} ${p.name}` : p.name,
                        })),
                    ]}
                />
                <SelectInput
                    label={t("resources.link.plan")}
                    value={planId}
                    disabled={projectId === ""}
                    onChange={(event) => {
                        setPlanId(event.target.value);
                        setTaskId("");
                    }}
                    options={[
                        choose(t("resources.link.plan")),
                        ...(plans.data ?? []).map((p) => ({
                            value: p.id,
                            label: p.name,
                        })),
                    ]}
                />
                <SelectInput
                    label={t("resources.link.task")}
                    value={taskId}
                    disabled={planId === ""}
                    onChange={(event) => setTaskId(event.target.value)}
                    options={[
                        choose(t("resources.link.task")),
                        ...(tasks.data ?? []).map((task) => ({
                            value: task.id,
                            label: task.title,
                        })),
                    ]}
                />
            </div>
            {errorMessage && (
                <p role="alert" className="mt-3 text-sm text-red-600">
                    {errorMessage}
                </p>
            )}
            <div className="mt-6 flex justify-end gap-2">
                <button
                    type="button"
                    onClick={onClose}
                    className="cursor-pointer rounded-md border border-gray-300 px-4 py-2 text-sm hover:bg-gray-100"
                >
                    {t("common.cancel")}
                </button>
                <button
                    type="button"
                    disabled={taskId === "" || isPending}
                    onClick={() => onPick(taskId)}
                    className="cursor-pointer rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:bg-accent/85 disabled:cursor-not-allowed disabled:opacity-50"
                >
                    {t("resources.link.submit")}
                </button>
            </div>
        </Modal>
    );
}
