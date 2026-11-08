import { useTranslate } from "../hooks/useTranslate";
import type { Plan } from "../types/plan";
import { PlanCard } from "./PlanCard";

type PlanListProps = {
    plans: Plan[];
    isLoading: boolean;
    isError: boolean;
};

export function PlanList({ plans, isError, isLoading }: PlanListProps) {
    const t = useTranslate();

    return (
        <section>
            <h2 className="mb-4 text-lg font-semibold">
                {t("project.plans.title")}
            </h2>
            {isLoading && (
                <p className="text-sm text-gray-500">{t("common.loading")}</p>
            )}
            {isError && (
                <p className="text-sm text-red-600">{t("common.error")}</p>
            )}
            {!isLoading && !isError && plans.length === 0 && (
                <p className="text-sm text-gray-500">
                    {t("project.plans.empty")}
                </p>
            )}
            {plans.length > 0 && (
                <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
                    {plans.map((plan) => (
                        <PlanCard key={plan.id} plan={plan} />
                    ))}
                </div>
            )}
        </section>
    );
}
