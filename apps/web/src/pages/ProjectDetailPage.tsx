import { useParams } from "react-router";
import { useTranslate } from "../hooks/useTranslate";
import { useProject, useProjectPlans } from "../hooks/useProjects";
import { ApiError } from "../api/client";
import NotFoundPage from "./NotFoundPage";
import { Sidebar } from "../components/SideBar";
import { TopBar } from "../components/TopBar";
import { ProjectHeader } from "../components/ProjectHeader";
import { PlanList } from "../components/PlanList";

export default function ProjectDetailPage() {
    const t = useTranslate();
    const { projectId = "" } = useParams();
    const project = useProject(projectId);
    const plans = useProjectPlans(projectId);

    if (project.error instanceof ApiError && project.error.status === 404)
        return <NotFoundPage />
    
    const path = [t("sidebar.projects"), project.data?.name ?? ""].filter(
        Boolean,
    )
    const planItems = plans.data ?? [];

    return (
        <div className="flex min-h-screen">
            <Sidebar activeHref="/projects" />
            <div className="flex flex-1 flex-col">
                <TopBar path={path} />
                <main className="flex-1 p-8">
                    <ProjectHeader
                        project={project.data}
                        planCount={planItems.length}
                        isLoading={project.isLoading}
                        isError={project.isError}
                    />
                    <PlanList
                        plans={planItems}
                        isLoading={plans.isLoading}
                        isError={plans.isError}
                    />
                </main>
            </div>
        </div>
    );
}
