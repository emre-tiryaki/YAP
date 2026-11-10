import { ApiError } from "../api/client";
import { useDeleteTask, useUpdateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import { ConfirmDialog } from "./ConfirmDialog";
import { EditTaskDialog, type EditableTask } from "./EditTaskDialog";
import { RenameDialog } from "./RenameDialog";

type TaskDialogsProps = {
    task: EditableTask & { id: string };
    action: CardAction | null;
    onClose: () => void;
};

export function TaskDialogs({ task, action, onClose }: TaskDialogsProps) {
    const t = useTranslate();
    const update = useUpdateTask();
    const remove = useDeleteTask();

    function close() {
        update.reset();
        remove.reset();
        onClose();
    }

    const updateError = !update.error
        ? null
        : update.error instanceof ApiError &&
            update.error.message === "invalid_input"
          ? t("plan.task.create.invalid")
          : t("common.error");

    return (
        <>
            <RenameDialog
                isOpen={action === "rename"}
                title={t("task.rename.title")}
                label={t("form.title")}
                initialName={task.title}
                isPending={update.isPending}
                errorMessage={updateError}
                onSubmit={(title) =>
                    update.mutate(
                        { taskId: task.id, input: { title } },
                        { onSuccess: close },
                    )
                }
                onClose={close}
            />
            <EditTaskDialog
                isOpen={action === "edit"}
                task={task}
                isPending={update.isPending}
                errorMessage={updateError}
                onSubmit={(input) =>
                    update.mutate(
                        { taskId: task.id, input },
                        { onSuccess: close },
                    )
                }
                onClose={close}
            />
            <ConfirmDialog
                isOpen={action === "delete"}
                title={t("plan.task.delete.title")}
                message={`"${task.title}" — ${t("plan.task.delete.message")}`}
                confirmLabel={t("plan.task.delete.confirm")}
                isPending={remove.isPending}
                isError={remove.isError}
                onConfirm={() => remove.mutate(task.id, { onSuccess: close })}
                onClose={close}
            />
        </>
    );
}
