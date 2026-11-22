import { useState } from "react";
import { FiCheckCircle, FiCircle, FiPlus, FiX } from "react-icons/fi";
import { Link } from "react-router";
import { ApiError } from "../api/client";
import { useProjectIdentifier } from "../hooks/useProjects";
import {
    useAddRelation,
    useRemoveRelation,
    useTaskRelations,
} from "../hooks/useTaskActivity";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { RelationView, TaskRelation } from "../types/taskActivity";
import { taskKey } from "../util/taskKey";
import { Button } from "./Button";
import { InlineError } from "./InlineError";
import { SelectInput } from "./SelectInput";
import { Skeleton } from "./Skeleton";
import { TaskPickerDialog } from "./TaskPickerDialog";

const groups: readonly RelationView[] = [
    "blocking",
    "blocked_by",
    "relates",
    "duplicate_of",
    "duplicated_by",
];

// The kinds a new relation can have, as the open task reads them.
const addable: readonly RelationView[] = [
    "blocking",
    "blocked_by",
    "relates",
    "duplicate_of",
];

export function TaskRelations({ taskId }: { taskId: string }) {
    const t = useTranslate();
    const relations = useTaskRelations(taskId);
    const add = useAddRelation(taskId);
    const [isAdding, setIsAdding] = useState(false);
    const [view, setView] = useState<RelationView>("blocking");

    function close() {
        setIsAdding(false);
        add.reset();
    }

    const items = relations.data ?? [];

    return (
        <div className="flex flex-col gap-3">
            {relations.isLoading && <Skeleton className="h-10" />}
            {relations.isError && (
                <InlineError
                    message={t("task.relations.loadError")}
                    onRetry={() => void relations.refetch()}
                />
            )}
            {relations.isSuccess && items.length === 0 && (
                <p className="text-sm text-fg-subtle">{t("task.relations.empty")}</p>
            )}
            {groups.map((group) => {
                const inGroup = items.filter((rel) => rel.view === group);
                if (inGroup.length === 0) return null;
                return (
                    <div key={group}>
                        <h4 className="mb-1 text-caption font-medium text-fg-subtle">
                            {t(`task.relation.${group}` as TranslationKey)}
                        </h4>
                        <ul className="flex flex-col rounded-md border border-line">
                            {inGroup.map((rel) => (
                                <RelationRow
                                    key={rel.taskId}
                                    taskId={taskId}
                                    relation={rel}
                                />
                            ))}
                        </ul>
                    </div>
                );
            })}
            <Button
                variant="secondary"
                size="md"
                leadingIcon={FiPlus}
                className="self-start"
                onClick={() => setIsAdding(true)}
            >
                {t("task.relations.add")}
            </Button>

            {isAdding && (
                <TaskPickerDialog
                    title={t("task.relations.add")}
                    submitLabel={t("task.relations.submit")}
                    excludeTaskId={taskId}
                    isPending={add.isPending}
                    errorMessage={
                        add.isError
                            ? add.error instanceof ApiError &&
                              add.error.message === "conflict"
                                ? t("task.relations.exists")
                                : t("task.relations.error")
                            : null
                    }
                    extra={
                        <SelectInput
                            label={t("task.relations.kind")}
                            value={view}
                            onChange={(event) =>
                                setView(event.target.value as RelationView)
                            }
                            options={addable.map((value) => ({
                                value,
                                label: t(`task.relation.${value}` as TranslationKey),
                            }))}
                        />
                    }
                    onPick={(relatedId) =>
                        add.mutate({ relatedId, view }, { onSuccess: close })
                    }
                    onClose={close}
                />
            )}
        </div>
    );
}

function RelationRow({
    taskId,
    relation,
}: {
    taskId: string;
    relation: TaskRelation;
}) {
    const t = useTranslate();
    const remove = useRemoveRelation(taskId);
    const key = taskKey(useProjectIdentifier(relation.projectId), relation.number);
    const isDone = relation.completedAt !== null;
    const DoneIcon = isDone ? FiCheckCircle : FiCircle;

    return (
        <li className="group flex items-center gap-2 border-b border-line px-3 py-1.5 text-body-xs last:border-b-0">
            <DoneIcon
                aria-hidden
                className={`shrink-0 ${isDone ? "text-success" : "text-fg-placeholder"}`}
            />
            {key && <span className="shrink-0 text-caption text-fg-subtle">{key}</span>}
            <Link
                to={`/projects/${relation.projectId}/plans/${relation.planId}?task=${relation.taskId}`}
                className={`min-w-0 flex-1 truncate hover:underline ${
                    isDone ? "text-fg-subtle line-through" : "text-fg"
                }`}
            >
                {relation.title}
            </Link>
            <button
                type="button"
                disabled={remove.isPending}
                onClick={() => remove.mutate(relation.taskId)}
                aria-label={`${t("task.relations.remove")}: ${relation.title}`}
                title={t("task.relations.remove")}
                className="cursor-pointer rounded p-1 text-fg-placeholder opacity-0 group-focus-within:opacity-100 group-hover:opacity-100 hover:bg-layer-1-hover hover:text-fg focus:opacity-100"
            >
                <FiX aria-hidden />
            </button>
        </li>
    );
}
