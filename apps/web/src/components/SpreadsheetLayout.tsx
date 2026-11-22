import { useState, type KeyboardEvent } from "react";
import { FiArrowDown, FiArrowUp, FiChevronDown } from "react-icons/fi";
import { useChangeTaskStatus } from "../hooks/useChangeTaskStatus";
import { useDateFormat } from "../hooks/useDateFormat";
import { useProjectIdentifier } from "../hooks/useProjects";
import { useUpdateTask } from "../hooks/useTasks";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import type { PlanTask, TaskPropertyKey } from "../types/task";
import type { DisplayOptions, OrderBy } from "../types/taskView";
import { fromDateInputValue, toDateInputValue } from "../util/date";
import { taskKey } from "../util/taskKey";
import { allTaskPropertyKeys } from "../util/taskView";
import { DatePicker } from "./DatePicker";
import { LabelsSelect } from "./LabelsSelect";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";
import { PrioritySelect } from "./PrioritySelect";
import { PropertySelect } from "./PropertySelect";

type SpreadsheetLayoutProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    // Filtered and ordered.
    tasks: PlanTask[];
    display: DisplayOptions;
    onChangeDisplay: (patch: Partial<DisplayOptions>) => void;
    onOpenTask: (task: PlanTask) => void;
};

const columnTitles: Record<TaskPropertyKey, TranslationKey> = {
    status: "task.property.status",
    priority: "task.property.priority",
    labels: "task.property.labels",
    startDate: "task.property.startDate",
    dueDate: "task.property.dueDate",
    subtasks: "display.property.subtasks",
    comments: "display.property.comments",
    createdAt: "display.property.createdAt",
    updatedAt: "display.property.updatedAt",
};

// Which columns the header menu can sort by.
const sortKeys: Partial<Record<TaskPropertyKey | "title", OrderBy>> = {
    title: "title",
    priority: "priority",
    dueDate: "due",
    createdAt: "created",
    updatedAt: "updated",
};

function SortHeader({
    id,
    label,
    display,
    onChangeDisplay,
}: {
    id: TaskPropertyKey | "title";
    label: string;
    display: DisplayOptions;
    onChangeDisplay: (patch: Partial<DisplayOptions>) => void;
}) {
    const t = useTranslate();
    const orderBy = sortKeys[id];
    if (!orderBy) return <>{label}</>;
    const active = display.orderBy === orderBy;
    return (
        <Menu
            trigger={(props) => (
                <button
                    type="button"
                    {...props}
                    className="flex w-full cursor-pointer items-center gap-1 text-left font-medium hover:text-fg"
                >
                    {label}
                    {active &&
                        (display.orderDir === "asc" ? (
                            <FiArrowUp aria-label={t("table.sortedBy")} />
                        ) : (
                            <FiArrowDown aria-label={t("table.sortedBy")} />
                        ))}
                    <FiChevronDown aria-hidden className="ml-auto text-fg-placeholder" />
                </button>
            )}
        >
            <MenuItem
                icon={<FiArrowUp />}
                label={t("table.sort.asc")}
                onSelect={() => onChangeDisplay({ orderBy, orderDir: "asc" })}
            />
            <MenuItem
                icon={<FiArrowDown />}
                label={t("table.sort.desc")}
                onSelect={() => onChangeDisplay({ orderBy, orderDir: "desc" })}
            />
        </Menu>
    );
}

export function SpreadsheetLayout({
    projectId,
    planId,
    statuses,
    tasks,
    display,
    onChangeDisplay,
    onOpenTask,
}: SpreadsheetLayoutProps) {
    const t = useTranslate();
    const columns = allTaskPropertyKeys.filter((key) =>
        display.properties.includes(key),
    );
    // Column 0 is the title; the others follow the display properties.
    const columnCount = columns.length + 1;
    const [focus, setFocus] = useState({ row: 0, col: 0 });

    function moveFocus(event: KeyboardEvent<HTMLTableSectionElement>) {
        const cell = event.target;
        // Only when the cell itself has focus; keys inside a select are its own.
        if (!(cell instanceof HTMLElement) || cell.dataset.cell !== "true") return;
        let { row, col } = focus;
        if (event.key === "ArrowDown") row = Math.min(row + 1, tasks.length - 1);
        else if (event.key === "ArrowUp") row = Math.max(row - 1, 0);
        else if (event.key === "ArrowRight") col = Math.min(col + 1, columnCount - 1);
        else if (event.key === "ArrowLeft") col = Math.max(col - 1, 0);
        else if (event.key === "Enter") {
            event.preventDefault();
            cell.querySelector<HTMLElement>("button, input")?.click();
            return;
        } else return;
        event.preventDefault();
        setFocus({ row, col });
        event.currentTarget
            .querySelector<HTMLElement>(`[data-pos="${row}:${col}"]`)
            ?.focus();
    }

    return (
        <div className="min-h-0 flex-1 overflow-auto rounded-lg border border-line">
            <table aria-label={t("table.label")} className="min-w-max border-collapse">
                <thead className="sticky top-0 z-20 bg-layer-1">
                    <tr className="h-9 border-b border-line text-body-xs text-fg-subtle">
                        <th
                            scope="col"
                            className="sticky left-0 z-30 w-80 min-w-80 bg-layer-1 px-3 text-left"
                        >
                            <SortHeader id="title" label={t("form.title")} display={display} onChangeDisplay={onChangeDisplay} />
                        </th>
                        {columns.map((key) => (
                            <th
                                key={key}
                                scope="col"
                                className="min-w-36 border-l border-line px-3 text-left"
                            >
                                <SortHeader id={key} label={t(columnTitles[key])} display={display} onChangeDisplay={onChangeDisplay} />
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody onKeyDown={moveFocus}>
                    {tasks.map((task, row) => (
                        <Row
                            key={task.id}
                            task={task}
                            row={row}
                            columns={columns}
                            focus={focus}
                            onFocusCell={(col) => setFocus({ row, col })}
                            projectId={projectId}
                            planId={planId}
                            statuses={statuses}
                            onOpenTask={onOpenTask}
                        />
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function Row({
    task,
    row,
    columns,
    focus,
    onFocusCell,
    projectId,
    planId,
    statuses,
    onOpenTask,
}: {
    task: PlanTask;
    row: number;
    columns: TaskPropertyKey[];
    focus: { row: number; col: number };
    onFocusCell: (col: number) => void;
    projectId: string;
    planId: string;
    statuses: Status[];
    onOpenTask: (task: PlanTask) => void;
}) {
    const t = useTranslate();
    const format = useDateFormat();
    const update = useUpdateTask();
    const changeStatus = useChangeTaskStatus(projectId, planId, statuses);
    const key = taskKey(useProjectIdentifier(projectId), task.number);
    const isTemp = task.id.startsWith("temp-");
    const isDone = task.completedAt !== null;

    function cellProps(col: number) {
        const active = focus.row === row && focus.col === col;
        return {
            "data-cell": "true",
            "data-pos": `${row}:${col}`,
            tabIndex: active ? 0 : -1,
            onFocus: () => onFocusCell(col),
            className:
                "h-10 border-l border-line p-0 text-body-xs text-fg-muted focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent",
        };
    }

    function render(keyName: TaskPropertyKey) {
        if (isTemp) return null;
        switch (keyName) {
            case "status":
                return (
                    <PropertySelect
                        variant="cell"
                        ariaLabel={task.title}
                        placeholder=""
                        value={task.statusId}
                        triggerLabel={
                            statuses.find((s) => s.id === task.statusId)?.title
                        }
                        onChange={(value) =>
                            typeof value === "string" &&
                            changeStatus.change(task, value)
                        }
                        options={statuses.map((s) => ({
                            value: s.id,
                            label: s.title,
                            icon: (
                                <span
                                    aria-hidden
                                    className="size-2.5 rounded-full"
                                    style={{ backgroundColor: s.color }}
                                />
                            ),
                        }))}
                    />
                );
            case "priority":
                return (
                    <PrioritySelect
                        variant="cell"
                        value={task.priority}
                        onChange={(priority) =>
                            update.mutate({ taskId: task.id, input: { priority } })
                        }
                    />
                );
            case "labels":
                return (
                    <LabelsSelect
                        variant="cell"
                        projectId={projectId}
                        value={task.labelIds}
                        onChange={(labelIds) =>
                            update.mutate({ taskId: task.id, input: { labelIds } })
                        }
                    />
                );
            case "startDate":
            case "dueDate": {
                const field = keyName === "startDate" ? "startDate" : "dueDate";
                return (
                    <div className="px-2">
                        <DatePicker
                            label={t(
                                keyName === "startDate"
                                    ? "form.startDate"
                                    : "form.dueDate",
                            )}
                            value={toDateInputValue(task[field])}
                            pill={{
                                overdue:
                                    field === "dueDate" &&
                                    task.dueDate !== null &&
                                    !isDone &&
                                    task.dueDate < new Date(new Date().setHours(0, 0, 0, 0)),
                            }}
                            onChange={(value) =>
                                update.mutate({
                                    taskId: task.id,
                                    input: { [field]: fromDateInputValue(value) },
                                })
                            }
                        />
                    </div>
                );
            }
            case "subtasks":
                return (
                    <span className="px-3">
                        {task.subtaskCount > 0
                            ? `${task.doneSubtaskCount}/${task.subtaskCount}`
                            : ""}
                    </span>
                );
            case "comments":
                return (
                    <span className="px-3">
                        {task.commentCount > 0 ? task.commentCount : ""}
                    </span>
                );
            case "createdAt":
                return <span className="px-3">{format.date(task.createdAt)}</span>;
            case "updatedAt":
                return <span className="px-3">{format.date(task.updatedAt)}</span>;
        }
    }

    return (
        <tr className="h-10 border-b border-line text-body-xs hover:bg-layer-1-hover">
            <td
                {...cellProps(0)}
                className={`sticky left-0 z-10 w-80 min-w-80 bg-surface-1 p-0 text-body-xs focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent`}
            >
                <button
                    type="button"
                    disabled={isTemp}
                    onClick={() => onOpenTask(task)}
                    className="flex h-10 w-full cursor-pointer items-center gap-2 px-3 text-left"
                >
                    {key && (
                        <span className="shrink-0 text-caption text-fg-subtle">
                            {key}
                        </span>
                    )}
                    <span
                        className={`truncate font-medium ${
                            isDone ? "text-fg-subtle line-through" : "text-fg"
                        }`}
                    >
                        {task.title}
                    </span>
                </button>
            </td>
            {columns.map((keyName, index) => (
                <td key={keyName} {...cellProps(index + 1)}>
                    {render(keyName)}
                </td>
            ))}
        </tr>
    );
}
