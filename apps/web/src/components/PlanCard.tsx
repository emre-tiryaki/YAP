import { useState } from "react";
import { FiCheckCircle, FiUsers } from "react-icons/fi";
import { Link } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Plan } from "../types/plan";
import { accentFor, initials } from "../util/accent";
import { CardMenu } from "./CardMenu";
import { PlanDialogs } from "./PlanDialogs";

type PlanCardProps = {
    plan: Plan;
};

export function PlanCard({ plan }: PlanCardProps) {
    const t = useTranslate();
    const [action, setAction] = useState<CardAction | null>(null);
    const accent = accentFor(plan.id);
    const progress =
        plan.taskCount === 0
            ? 0
            : Math.round((plan.doneTaskCount / plan.taskCount) * 100);
    const isComplete = plan.taskCount > 0 && progress === 100;

    return (
        <div className="relative h-full">
            <Link
                to={`/projects/${plan.projectId}/plans/${plan.id}`}
                className={`flex h-full flex-col rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition duration-200 hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-2 focus-visible:outline-accent ${accent.hoverBorder}`}
            >
                <div className="flex items-start gap-3 pr-8">
                    <span
                        aria-hidden
                        className={`grid size-10 shrink-0 place-items-center rounded-lg font-semibold ${
                            plan.icon ? "text-2xl" : `text-sm ${accent.tile}`
                        }`}
                    >
                        {plan.icon || initials(plan.name)}
                    </span>
                    <div className="min-w-0 flex-1">
                        <h3
                            className="truncate leading-tight font-semibold text-gray-900"
                            title={plan.name}
                        >
                            {plan.name}
                        </h3>
                        {plan.teamName && (
                            <span className="mt-1 inline-flex max-w-full items-center gap-1 truncate rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
                                <FiUsers aria-hidden className="shrink-0" />
                                {plan.teamName}
                            </span>
                        )}
                    </div>
                </div>

                <p
                    title={plan.description || undefined}
                    className={`mt-3 line-clamp-2 min-h-10 flex-1 text-sm break-words ${
                        plan.description
                            ? "text-gray-600"
                            : "text-gray-400 italic"
                    }`}
                >
                    {plan.description || t("project.noDescription")}
                </p>

                <div className="mt-4">
                    <div className="mb-1.5 flex items-center justify-between text-xs">
                        <span className="text-gray-500">
                            <span className="font-semibold text-gray-900">
                                {plan.doneTaskCount}
                            </span>
                            /{plan.taskCount} {t("plan.stats.tasks")}{" "}
                            {t("plan.stats.done")}
                        </span>
                        {isComplete ? (
                            <span className="inline-flex items-center gap-1 font-medium text-emerald-600">
                                <FiCheckCircle aria-hidden />
                                {progress}%
                            </span>
                        ) : (
                            <span className="font-medium text-gray-500 tabular-nums">
                                {progress}%
                            </span>
                        )}
                    </div>
                    <div
                        role="progressbar"
                        aria-valuemin={0}
                        aria-valuemax={100}
                        aria-valuenow={progress}
                        aria-label={t("plan.stats.progress")}
                        className="h-1.5 overflow-hidden rounded-full bg-gray-100"
                    >
                        <div
                            className={`h-full rounded-full transition-[width] duration-500 ${
                                isComplete ? "bg-emerald-500" : accent.bar
                            }`}
                            style={{ width: `${progress}%` }}
                        />
                    </div>
                </div>
            </Link>
            <div className="absolute top-3 right-3">
                <CardMenu onSelect={setAction} />
            </div>
            <PlanDialogs
                plan={plan}
                action={action}
                onClose={() => setAction(null)}
            />
        </div>
    );
}
