import { BrowserRouter, Routes, Route, Navigate, Outlet, useLocation } from "react-router"
import WorkspaceHomePage from "./pages/WorkspaceHomePage"
import NotFoundPage from "./pages/NotFoundPage";
import ProjectDetailPage from "./pages/ProjectDetailPage";
import PlanDetailPage from "./pages/PlanDetailPage";
import ResourcesPage from "./pages/ResourcesPage";
import TaskPage from "./pages/TaskPage";
import ProjectsPage from "./pages/ProjectsPage";
import UpcomingTasksPage from "./pages/UpcomingTasksPage";
import { useEffect } from "react";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { CommandPalette } from "./components/CommandPalette";
import { GlobalShortcuts } from "./components/GlobalShortcuts";
import { SettingsModal } from "./components/SettingsModal";
import { ShortcutsDialog } from "./components/ShortcutsDialog";
import { Sidebar } from "./components/SideBar";
import { closeSettings, useSettingsDialog } from "./hooks/useSettingsDialog";
import { NotificationCenter } from "./components/NotificationCenter";
import { usePreferences } from "./hooks/usePreferences";
import { useTheme } from "./hooks/useTheme";

// Shared shell: a grey canvas with the sidebar on it and the page in a
// rounded frame. The sidebar stays mounted across navigations; the frame
// (`app-frame`) turns translucent over a background picture.
function AppLayout() {
    const { pathname } = useLocation();
    const settingsSection = useSettingsDialog();
    return (
        <div className="app-canvas flex h-screen gap-2 bg-canvas p-2">
            <Sidebar />
            <div className="app-frame flex min-w-0 flex-1 flex-col overflow-hidden rounded-lg border border-line bg-surface-1">
                <ErrorBoundary key={pathname}>
                    <Outlet />
                </ErrorBoundary>
            </div>
            <NotificationCenter />
            <GlobalShortcuts />
            <CommandPalette />
            <ShortcutsDialog />
            <SettingsModal
                isOpen={settingsSection !== null}
                initialSection={settingsSection ?? undefined}
                onClose={closeSettings}
            />
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
                  <Route path="/projects/:projectId/plans/:planId/tasks/:taskId" element={<TaskPage />} />
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
