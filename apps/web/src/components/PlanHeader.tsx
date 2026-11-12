import { useState } from "react";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Plan } from "../types/plan";
import { CardMenu } from "./CardMenu";
import { PlanDialogs } from "./PlanDialogs";
import { PlanIconButton } from "./PlanIconButton";

type PlanHeaderProps = {
    plan: Plan;
    onDeleted: () => void;
};

export function PlanHeader({ plan, onDeleted }: PlanHeaderProps) {
    const t = useTranslate();
    const [action, setAction] = useState<CardAction | null>(null);
    const percent =
        plan.taskCount === 0
            ? 0
            : Math.round((plan.doneTaskCount / plan.taskCount) * 100);

    return (
        <header className="mb-6 rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
            <div className="flex items-start justify-between gap-3">
                <div className="flex min-w-0 items-center gap-3">
                    <PlanIconButton plan={plan} />
                    <h1 className="min-w-0 text-2xl font-bold break-words">
                        {plan.name}
                    </h1>
                    {plan.teamName && (
                        <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-700">
                            {plan.teamName}
                        </span>
                    )}
                </div>
                <CardMenu onSelect={setAction} />
            </div>
            {plan.description ? (
                <p className="mt-2 text-sm text-gray-600">{plan.description}</p>
            ) : (
                <p className="mt-2 text-sm text-gray-400 italic">
                    {t("project.noDescription")}
                </p>
            )}
            <dl className="mt-4 flex gap-6 text-xs text-gray-500">
                <div className="flex gap-1">
                    <dt className="font-semibold text-gray-900">
                        {plan.taskCount}
                    </dt>
                    <dd>{t("plan.stats.tasks")}</dd>
                </div>
                <div className="flex gap-1">
                    <dt className="font-semibold text-gray-900">
                        {plan.doneTaskCount}
                    </dt>
                    <dd>{t("plan.stats.done")}</dd>
                </div>
            </dl>
            <div className="mt-3 flex items-center gap-3">
                <div
                    role="progressbar"
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-valuenow={percent}
                    className="h-2 flex-1 overflow-hidden rounded-full bg-gray-100"
                >
                    <div
                        className="h-full rounded-full bg-green-500 transition-all"
                        style={{ width: `${percent}%` }}
                    />
                </div>
                <span className="text-xs text-gray-500">
                    %{percent} {t("plan.stats.progress")}
                </span>
            </div>
            <PlanDialogs
                plan={plan}
                action={action}
                onClose={() => setAction(null)}
                onDeleted={onDeleted}
            />
        </header>
    );
}

export function PlanHeaderSkeleton() {
    return (
        <div
            aria-busy="true"
            className="mb-6 animate-pulse rounded-lg border border-gray-200 bg-white p-6 shadow-sm"
        >
            <div className="h-7 w-1/3 rounded bg-gray-200" />
            <div className="mt-3 h-4 w-2/3 rounded bg-gray-100" />
            <div className="mt-5 h-3 w-40 rounded bg-gray-100" />
            <div className="mt-4 h-2 w-full rounded bg-gray-100" />
        </div>
    );
}
