import { useUpdatePlan } from "../hooks/useProjects";
import type { Plan } from "../types/plan";
import { IconButton } from "./IconButton";

type PlanIconButtonProps = {
    plan: Plan;
};

export function PlanIconButton({ plan }: PlanIconButtonProps) {
    const update = useUpdatePlan(plan.projectId);
    return (
        <IconButton
            id={plan.id}
            name={plan.name}
            icon={plan.icon}
            error={update.error}
            reset={update.reset}
            onSelect={(icon, onSaved) =>
                update.mutate(
                    { planId: plan.id, input: { icon } },
                    { onSuccess: onSaved },
                )
            }
        />
    );
}
