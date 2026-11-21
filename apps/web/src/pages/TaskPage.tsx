import { useParams } from "react-router";
import { PageHeader } from "../components/PageHeader";
import { Skeleton } from "../components/Skeleton";
import { TaskBody } from "../components/TaskBody";
import { usePlan, usePlanStatuses } from "../hooks/usePlans";
import { useProject } from "../hooks/useProjects";
import { usePlanTasks } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { BreadcrumbItem } from "../types/breadcrumb";
import { taskKey } from "../util/taskKey";
import NotFoundPage from "./NotFoundPage";

// A task on its own page (/projects/:projectId/plans/:planId/tasks/:taskId):
// the peek's content in the full layout.
export default function TaskPage() {
    const t = useTranslate();
    const { projectId = "", planId = "", taskId = "" } = useParams();
    const project = useProject(projectId);
    const plan = usePlan(projectId, planId);
    const statuses = usePlanStatuses(projectId, planId);
    const tasks = usePlanTasks(projectId, planId);
    const task = tasks.data?.find((other) => other.id === taskId);

    if (tasks.isSuccess && !task) return <NotFoundPage />;

    const breadcrumb: BreadcrumbItem[] = [
        { label: t("sidebar.projects"), to: "/projects" },
        { label: project.data?.name ?? "", to: `/projects/${projectId}` },
        {
            label: plan.data?.name ?? "",
            to: `/projects/${projectId}/plans/${planId}`,
        },
        {
            label: task
                ? `${taskKey(project.data?.identifier ?? "", task.number)} ${task.title}`.trim()
                : "",
        },
    ].filter((item) => item.label);

    return (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <PageHeader breadcrumb={breadcrumb} />
            <div className="min-h-0 flex-1 overflow-auto">
                {task && statuses.data ? (
                    <div className="mx-auto max-w-6xl">
                        <TaskBody
                            key={task.id}
                            projectId={projectId}
                            planId={planId}
                            task={task}
                            statuses={statuses.data}
                            full
                        />
                    </div>
                ) : (
                    <div aria-busy="true" className="mx-auto flex max-w-6xl flex-col gap-4 px-8 py-5">
                        <Skeleton className="h-6 w-1/3" />
                        <Skeleton className="h-40" />
                    </div>
                )}
            </div>
        </div>
    );
}
