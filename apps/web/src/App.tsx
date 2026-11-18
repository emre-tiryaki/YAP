import { BrowserRouter, Routes, Route, Navigate, Outlet } from "react-router"
import WorkspaceHomePage from "./pages/WorkspaceHomePage"
import NotFoundPage from "./pages/NotFoundPage";
import ProjectDetailPage from "./pages/ProjectDetailPage";
import PlanDetailPage from "./pages/PlanDetailPage";
import ResourcesPage from "./pages/ResourcesPage";
import ProjectsPage from "./pages/ProjectsPage";
import UpcomingTasksPage from "./pages/UpcomingTasksPage";
import { useEffect } from "react";
import { Sidebar } from "./components/SideBar";
import { NotificationCenter } from "./components/NotificationCenter";
import { usePreferences } from "./hooks/usePreferences";
import { useTheme } from "./hooks/useTheme";

// Shared shell: the sidebar stays mounted across page navigations. The page
// sits on `app-surface`, which turns translucent over a background picture.
function AppLayout() {
  return (
      <div className="flex min-h-screen">
          <Sidebar />
          <div className="app-surface flex min-w-0 flex-1">
              <Outlet />
          </div>
          <NotificationCenter />
      </div>
  );
}

function App() {
    const { language } = usePreferences();
    useEffect(() => {
        document.documentElement.lang = language;
    }, [language]);
    useTheme();

  return (
      <BrowserRouter>
          <Routes>
              <Route path="/" element={<Navigate to={"/home"} replace />} />
              <Route element={<AppLayout />}>
                  <Route path="/home" element={<WorkspaceHomePage />} />
                  <Route path="/projects" element={<ProjectsPage />} />
                  <Route path="/tasks" element={<UpcomingTasksPage />} />
                  <Route path="/projects/:projectId" element={<ProjectDetailPage />} />
                  <Route path="/projects/:projectId/plans/:planId" element={<PlanDetailPage />} />
                  <Route path="/resources" element={<ResourcesPage />} />
                  <Route path="/resources/trash" element={<ResourcesPage view="trash" />} />
                  <Route path="/resources/:resourceId" element={<ResourcesPage />} />
              </Route>
              <Route path="*" element={<NotFoundPage />} />
          </Routes>
      </BrowserRouter>
  );
}

export default App
