import { ApiError } from "../api/client";
import { useDeleteProject, useUpdateProject } from "../hooks/useProjects";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Project } from "../types/project";
import { ConfirmDialog } from "./ConfirmDialog";
import { EditDetailsDialog } from "./EditDetailsDialog";
import { RenameDialog } from "./RenameDialog";

type ProjectDialogsProps = {
    project: Project;
    action: CardAction | null;
    onClose: () => void;
    // Called after a successful delete, e.g. to leave the deleted item's page.
    onDeleted?: () => void;
};

export function ProjectDialogs({
    project,
    action,
    onClose,
    onDeleted,
}: ProjectDialogsProps) {
    const t = useTranslate();
    const update = useUpdateProject();
    const remove = useDeleteProject();

    function close() {
        update.reset();
        remove.reset();
        onClose();
    }

    const updateError = !update.error
        ? null
        : update.error instanceof ApiError &&
            update.error.message === "invalid_input"
          ? t("project.invalid")
          : t("common.error");

    return (
        <>
            <RenameDialog
                isOpen={action === "rename"}
                title={t("project.rename.title")}
                label={t("form.name")}
                initialName={project.name}
                isPending={update.isPending}
                errorMessage={updateError}
                onSubmit={(name) =>
                    update.mutate(
                        { projectId: project.id, input: { name } },
                        { onSuccess: close },
                    )
                }
                onClose={close}
            />
            <EditDetailsDialog
                isOpen={action === "edit"}
                title={t("project.edit.title")}
                initial={project}
                withIcon
                isPending={update.isPending}
                errorMessage={updateError}
                onSubmit={(input) =>
                    update.mutate(
                        { projectId: project.id, input },
                        { onSuccess: close },
                    )
                }
                onClose={close}
            />
            <ConfirmDialog
                isOpen={action === "delete"}
                title={t("project.delete.title")}
                message={`"${project.name}" — ${t("project.delete.message")}`}
                confirmLabel={t("common.delete")}
                isPending={remove.isPending}
                isError={remove.isError}
                onConfirm={() =>
                    remove.mutate(project.id, {
                        onSuccess: () => {
                            close();
                            onDeleted?.();
                        },
                    })
                }
                onClose={close}
            />
        </>
    );
}
