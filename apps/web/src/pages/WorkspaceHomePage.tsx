import { ProjectList } from "../components/ProjectList";
import { Sidebar } from "../components/SideBar";
import { TaskList } from "../components/TaskList";
import { useTranslate } from "../hooks/useTranslate";
import { useWorkspaceHome } from "../hooks/useWorkspaceHome";

export default function WorkspaceHomePage() {
  const t = useTranslate();
  const { userName, projects, tasks } = useWorkspaceHome();

  return (
    <div className="flex min-h-screen">
      <Sidebar activeHref="/" />
      <main className="flex-1 p-8">
        <h1 className="mb-8 text-2xl font-bold">
          {t("workspace.welcome")}, {userName}
        </h1>
        <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
          <ProjectList projects={projects} />
          <TaskList tasks={tasks} />
        </div>
      </main>
    </div>
  );
}
