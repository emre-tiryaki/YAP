import { ApiError } from "../api/client";
import { useDeletePlan, useUpdatePlan } from "../hooks/useProjects";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Plan } from "../types/plan";
import { ConfirmDialog } from "./ConfirmDialog";
import { EditDetailsDialog } from "./EditDetailsDialog";
import { RenameDialog } from "./RenameDialog";

type PlanDialogsProps = {
    plan: Plan;
    action: CardAction | null;
    onClose: () => void;
    // Called after a successful delete, e.g. to leave the deleted item's page.
    onDeleted?: () => void;
};

function errorCode(error: Error | null): string | null {
    if (!error) return null;
    return error instanceof ApiError ? error.message : "unknown";
}

export function PlanDialogs({
    plan,
    action,
    onClose,
    onDeleted,
}: PlanDialogsProps) {
    const t = useTranslate();
    const update = useUpdatePlan(plan.projectId);
    const remove = useDeletePlan(plan.projectId);

    function close() {
        update.reset();
        remove.reset();
        onClose();
    }

    const updateCode = errorCode(update.error);
    const updateError =
        updateCode === null
            ? null
            : updateCode === "conflict"
              ? t("project.plans.create.duplicate")
              : updateCode === "invalid_input"
                ? t("project.plans.create.invalid")
                : t("common.error");

    return (
        <>
            <RenameDialog
                isOpen={action === "rename"}
                title={t("plan.rename.title")}
                label={t("form.name")}
                initialName={plan.name}
                isPending={update.isPending}
                errorMessage={updateError}
                onSubmit={(name) =>
                    update.mutate(
                        { planId: plan.id, input: { name } },
                        { onSuccess: close },
                    )
                }
                onClose={close}
            />
            <EditDetailsDialog
                isOpen={action === "edit"}
                title={t("plan.edit.title")}
                initial={plan}
                withIcon
                isPending={update.isPending}
                errorMessage={updateError}
                onSubmit={(input) =>
                    update.mutate(
                        { planId: plan.id, input },
                        { onSuccess: close },
                    )
                }
                onClose={close}
            />
            <ConfirmDialog
                isOpen={action === "delete"}
                title={t("plan.delete.title")}
                message={`"${plan.name}" — ${t("plan.delete.message")}`}
                confirmLabel={t("common.delete")}
                isPending={remove.isPending}
                isError={remove.isError}
                errorMessage={
                    errorCode(remove.error) === "conflict"
                        ? t("plan.delete.lastPlan")
                        : undefined
                }
                onConfirm={() =>
                    remove.mutate(plan.id, {
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
