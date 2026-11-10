import { Link } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import type { Plan } from "../types/plan";

type PlanCardProps = {
    plan: Plan;
};

export function PlanCard({ plan }: PlanCardProps) {
    const t = useTranslate();

    return (
        <Link
            to={`/projects/${plan.projectId}/plans/${plan.id}`}
            className="block rounded-lg border border-gray-200 bg-white p-4 shadow-sm transition hover:border-gray-400 hover:shadow-md"
        >
            <div className="flex items-center justify-between gap-2">
                <h3 className="font-semibold">{plan.name}</h3>
                <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-700">
                    {plan.teamName ?? t("project.plans.wholeProject")}
                </span>
            </div>
            {plan.description && (
                <p className="mt-1 text-sm text-gray-600">{plan.description}</p>
            )}
            <p className="mt-3 text-xs text-gray-500">
                {plan.taskCount} {t("project.plans.taskCount")}
            </p>
        </Link>
    );
}
