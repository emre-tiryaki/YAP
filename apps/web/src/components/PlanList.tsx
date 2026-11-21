import { useTranslate } from "../hooks/useTranslate";
import type { Plan } from "../types/plan";
import { InlineError } from "./InlineError";
import { PlanCard } from "./PlanCard";
import { SectionHeader } from "./SectionHeader";
import { Button } from "./Button";
import { FiLayout } from "react-icons/fi";
import { EmptyState } from "./EmptyState";
import { Skeleton } from "./Skeleton";

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
                        <Skeleton key={i} className="h-40 rounded-xl" />
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
                <EmptyState
                    icon={FiLayout}
                    title={t("project.plans.empty")}
                    description={t("project.plans.emptyHint")}
                    action={
                        <Button variant="primary" size="lg" onClick={onCreate}>
                            + {t("project.plans.create.title")}
                        </Button>
                    }
                />
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
