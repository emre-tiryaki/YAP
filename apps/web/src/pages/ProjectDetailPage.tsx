import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import { useProject, useProjectPlans } from "../hooks/useProjects";
import { ApiError } from "../api/client";
import NotFoundPage from "./NotFoundPage";
import { TopBar } from "../components/TopBar";
import type { BreadcrumbItem } from "../types/breadcrumb";
import {
    ProjectHeader,
    ProjectHeaderSkeleton,
} from "../components/ProjectHeader";
import { PlanList } from "../components/PlanList";
import { InlineError } from "../components/InlineError";
import { Modal } from "../components/Modal";
import { CreatePlanForm } from "../components/CreatePlanForm";

export default function ProjectDetailPage() {
    const t = useTranslate();
    const navigate = useNavigate();
    const { projectId = "" } = useParams();
    const project = useProject(projectId);
    const plans = useProjectPlans(projectId);
    const [isPlanModalOpen, setIsPlanModalOpen] = useState(false);

    if (project.error instanceof ApiError && project.error.status === 404)
        return <NotFoundPage />;

    const path: BreadcrumbItem[] = [
        { label: t("sidebar.projects"), href: "/home" },
        { label: project.data?.name ?? "", href: `/projects/${projectId}` },
    ].filter((item) => item.label);

    return (
        <div className="flex min-w-0 flex-1 flex-col">
            <TopBar path={path} />
            <main className="flex-1 p-8">
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
                        <PlanList
                            plans={plans.data ?? []}
                            isLoading={plans.isLoading}
                            isError={plans.isError}
                            onRetry={() => void plans.refetch()}
                            onCreate={() => setIsPlanModalOpen(true)}
                        />
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
