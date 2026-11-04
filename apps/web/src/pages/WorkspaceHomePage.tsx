import { Sidebar } from "../components/SideBar";
import { useTranslate } from "../hooks/useTranslate";
import { useWorkspace } from "../hooks/useWorkspace";
import { useUpcomingTasks } from "../hooks/useTasks";
import { useProjects } from "../hooks/useProjects";
import { ProjectList } from "../components/ProjectList";
import { CreateProjectForm } from "../components/CreateProjectForm";
import { TaskList } from "../components/TaskList";
import { CreateTaskForm } from "../components/CreateTaskForm";

export default function WorkspaceHomePage() {
  const t = useTranslate();
  const workspace = useWorkspace()
  const projects = useProjects()
  const tasks = useUpcomingTasks()

  const projectItems = projects.data ?? []

  return (
      <div className="flex min-h-screen">
          <Sidebar activeHref="/" />
          <main className="flex-1 p-8">
              <h1>
                  {t("workspace.welcome")}
                  {workspace.data && `, ${workspace.data.username}`}
              </h1>
              <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
                  <div className="flex flex-col gap-8">
                      <ProjectList
                          projects={projectItems}
                          isError={projects.isError}
                          isLoading={projects.isLoading}
                      />
                      <CreateProjectForm />
                  </div>
                  <div className="flex flex-col gap-8">
                      <TaskList
                          tasks={tasks.data ?? []}
                          isError={tasks.isError}
                          isLoading={tasks.isLoading}
                      />
                      <CreateTaskForm projects={projectItems} />
                  </div>
              </div>
          </main>
      </div>
  );
}
