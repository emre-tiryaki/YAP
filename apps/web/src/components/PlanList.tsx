import { useTranslate } from "../hooks/useTranslate";
import type { Plan } from "../types/plan";
import { InlineError } from "./InlineError";
import { PlanCard } from "./PlanCard";
import { SectionHeader } from "./SectionHeader";

type PlanListProps = {
    plans: Plan[];
    isLoading: boolean;
    isError: boolean;
    onRetry: () => void;
    onCreate: () => void;
};

const gridClass = "grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3";

export function PlanList({
    plans,
    isError,
    isLoading,
    onRetry,
    onCreate,
}: PlanListProps) {
    const t = useTranslate();

    return (
        <section>
            <SectionHeader
                title={t("project.plans.title")}
                onCreate={onCreate}
            />
            {isLoading && (
                <div aria-busy="true" className={gridClass}>
                    {[0, 1, 2].map((i) => (
                        <div
                            key={i}
                            className="h-40 animate-pulse rounded-xl border border-gray-200 bg-gray-100"
                        />
                    ))}
                </div>
            )}
            {isError && (
                <InlineError
                    message={t("project.plans.loadError")}
                    onRetry={onRetry}
                />
            )}
            {!isLoading && !isError && plans.length === 0 && (
                <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-gray-300 p-8 text-center">
                    <p className="text-sm font-medium text-gray-700">
                        {t("project.plans.empty")}
                    </p>
                    <p className="text-sm text-gray-500">
                        {t("project.plans.emptyHint")}
                    </p>
                    <button
                        type="button"
                        onClick={onCreate}
                        className="cursor-pointer rounded-md bg-gray-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-gray-700"
                    >
                        + {t("project.plans.create.title")}
                    </button>
                </div>
            )}
            {!isError && plans.length > 0 && (
                <div className={gridClass}>
                    {plans.map((plan) => (
                        <PlanCard key={plan.id} plan={plan} />
                    ))}
                </div>
            )}
        </section>
    );
}
