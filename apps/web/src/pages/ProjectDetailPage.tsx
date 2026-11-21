import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import { useProject, useProjectPlans } from "../hooks/useProjects";
import { ApiError } from "../api/client";
import NotFoundPage from "./NotFoundPage";
import { FiPlus } from "react-icons/fi";
import { HeaderAction } from "../components/HeaderAction";
import { PageHeader } from "../components/PageHeader";
import type { BreadcrumbItem } from "../types/breadcrumb";
import {
    ProjectHeader,
    ProjectHeaderSkeleton,
} from "../components/ProjectHeader";
import { PlanList } from "../components/PlanList";
import { InlineError } from "../components/InlineError";
import { Modal } from "../components/Modal";
import { CreatePlanForm } from "../components/CreatePlanForm";
import { ResourceActivity } from "../components/ResourceActivity";

export default function ProjectDetailPage() {
    const t = useTranslate();
    const navigate = useNavigate();
    const { projectId = "" } = useParams();
    const project = useProject(projectId);
    const plans = useProjectPlans(projectId);
    const [isPlanModalOpen, setIsPlanModalOpen] = useState(false);

    if (project.error instanceof ApiError && project.error.status === 404)
        return <NotFoundPage />;

    const breadcrumb: BreadcrumbItem[] = [
        { label: t("sidebar.projects"), to: "/projects" },
        { label: project.data?.name ?? "", to: `/projects/${projectId}` },
    ].filter((item) => item.label);

    return (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            <PageHeader
                breadcrumb={breadcrumb}
                actions={
                    <HeaderAction
                        variant="primary"
                        icon={FiPlus}
                        label={t("project.plans.create.title")}
                        onClick={() => setIsPlanModalOpen(true)}
                    />
                }
            />
            <main className="flex-1 overflow-auto p-8">
                {project.isLoading && <ProjectHeaderSkeleton />}
                {project.isError && (
                    <InlineError
                        message={t("project.loadError")}
                        onRetry={() => void project.refetch()}
                    />
                )}
                {project.data && (
                    <>
                        <ProjectHeader
                            project={project.data}
                            planCount={plans.data?.length}
                            onDeleted={() =>
                                void navigate("/home", { replace: true })
                            }
                        />
                        <div className="flex flex-col gap-8 xl:flex-row xl:items-start">
                            <div className="min-w-0 flex-1">
                                <PlanList
                                    plans={plans.data ?? []}
                                    isLoading={plans.isLoading}
                                    isError={plans.isError}
                                    onRetry={() => void plans.refetch()}
                                    onCreate={() => setIsPlanModalOpen(true)}
                                />
                            </div>
                            <aside
                                aria-labelledby="project-activity"
                                className="w-full shrink-0 rounded-lg border border-gray-200 bg-white p-5 xl:sticky xl:top-8 xl:w-80"
                            >
                                <h2 id="project-activity" className="mb-4 text-sm font-semibold text-gray-900">
                                    {t("project.activity.title")}
                                </h2>
                                <ResourceActivity
                                    scope={{ kind: "project", id: projectId }}
                                    emptyText={t("project.activity.empty")}
                                />
                            </aside>
                        </div>
                    </>
                )}
            </main>

            <Modal
                isOpen={isPlanModalOpen}
                title={t("project.plans.create.title")}
                onClose={() => setIsPlanModalOpen(false)}
            >
                <CreatePlanForm
                    projectId={projectId}
                    onDone={() => setIsPlanModalOpen(false)}
                />
            </Modal>
        </div>
    );
}
