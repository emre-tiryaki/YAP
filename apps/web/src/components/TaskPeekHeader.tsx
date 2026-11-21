import {
    FiCheck,
    FiChevronsRight,
    FiLink,
    FiMaximize2,
    FiMonitor,
    FiMoreHorizontal,
    FiSidebar,
    FiSquare,
    FiTrash2,
    FiX,
} from "react-icons/fi";
import { Link } from "react-router";
import { updatePreferences } from "../hooks/usePreferences";
import { useSetTaskDone } from "../hooks/useTasks";
import { pushInfoToast } from "../hooks/useToasts";
import { useTranslate } from "../hooks/useTranslate";
import type { TaskPeekMode } from "../types/settings";
import type { PlanTask } from "../types/task";
import { Button } from "./Button";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";
import { MenuSeparator } from "./MenuSeparator";
import { Tooltip } from "./Tooltip";

type TaskPeekHeaderProps = {
    task: PlanTask;
    projectId: string;
    planId: string;
    mode: TaskPeekMode;
    onClose: () => void;
    onDelete: () => void;
};

const modes = [
    { mode: "side", label: "task.peek.mode.side", Icon: FiSidebar },
    { mode: "modal", label: "task.peek.mode.modal", Icon: FiSquare },
    { mode: "full", label: "task.peek.mode.full", Icon: FiMonitor },
] as const;

export function TaskPeekHeader({
    task,
    projectId,
    planId,
    mode,
    onClose,
    onDelete,
}: TaskPeekHeaderProps) {
    const t = useTranslate();
    const setDone = useSetTaskDone(projectId, planId);
    const isDone = task.completedAt !== null;
    const fullPage = `/projects/${projectId}/plans/${planId}/tasks/${task.id}`;

    function copyLink() {
        const url = `${window.location.origin}/projects/${projectId}/plans/${planId}?task=${task.id}`;
        void navigator.clipboard.writeText(url);
        pushInfoToast(t("task.peek.linkCopied"), task.title);
    }

    return (
        <div className="flex h-12 shrink-0 items-center gap-1 border-b border-line px-4">
            <Tooltip label={t("task.peek.close")}>
                <Button
                    variant="ghost"
                    size="sm"
                    aria-label={t("task.peek.close")}
                    onClick={onClose}
                >
                    {mode === "side" ? (
                        <FiChevronsRight aria-hidden />
                    ) : (
                        <FiX aria-hidden />
                    )}
                </Button>
            </Tooltip>
            <Tooltip label={t("task.peek.openFull")}>
                <Link
                    to={fullPage}
                    aria-label={t("task.peek.openFull")}
                    className="inline-flex h-6 items-center rounded-md px-2 text-fg-muted hover:bg-layer-1-hover hover:text-fg focus-visible:outline-2 focus-visible:outline-accent"
                >
                    <FiMaximize2 aria-hidden />
                </Link>
            </Tooltip>
            <Menu
                trigger={({ ref, ...props }) => (
                    <Tooltip label={t("task.peek.mode")}>
                        <Button
                            ref={ref}
                            variant="ghost"
                            size="sm"
                            aria-label={t("task.peek.mode")}
                            {...props}
                        >
                            {modes.map(
                                ({ mode: m, Icon }) =>
                                    m === mode && <Icon key={m} aria-hidden />,
                            )}
                        </Button>
                    </Tooltip>
                )}
            >
                {modes.map(({ mode: m, label, Icon }) => (
                    <MenuItem
                        key={m}
                        icon={m === mode ? <FiCheck aria-hidden /> : <Icon aria-hidden />}
                        label={t(label)}
                        onSelect={() => updatePreferences({ taskPeekMode: m })}
                    />
                ))}
            </Menu>
            <Tooltip label={t("task.peek.copyLink")}>
                <Button
                    variant="ghost"
                    size="sm"
                    aria-label={t("task.peek.copyLink")}
                    onClick={copyLink}
                >
                    <FiLink aria-hidden />
                </Button>
            </Tooltip>
            <div className="flex-1" />
            <Menu
                placement="bottom-end"
                trigger={({ ref, ...props }) => (
                    <Button
                        ref={ref}
                        variant="ghost"
                        size="sm"
                        aria-label={t("task.peek.more")}
                        {...props}
                    >
                        <FiMoreHorizontal aria-hidden />
                    </Button>
                )}
            >
                <MenuItem
                    icon={<FiCheck aria-hidden />}
                    label={t(isDone ? "task.detail.markOpen" : "task.detail.markDone")}
                    onSelect={() =>
                        setDone.mutate({ taskId: task.id, done: !isDone })
                    }
                />
                <MenuSeparator />
                <MenuItem
                    danger
                    icon={<FiTrash2 aria-hidden />}
                    label={t("common.delete")}
                    onSelect={onDelete}
                />
            </Menu>
        </div>
    );
}
