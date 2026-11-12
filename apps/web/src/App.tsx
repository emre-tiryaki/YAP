import { BrowserRouter, Routes, Route, Navigate, Outlet } from "react-router"
import WorkspaceHomePage from "./pages/WorkspaceHomePage"
import NotFoundPage from "./pages/NotFoundPage";
import ProjectDetailPage from "./pages/ProjectDetailPage";
import PlanDetailPage from "./pages/PlanDetailPage";
import { Sidebar } from "./components/SideBar";

// Shared shell: the sidebar stays mounted across page navigations.
function AppLayout() {
  return (
      <div className="flex min-h-screen">
          <Sidebar />
          <Outlet />
      </div>
  );
}

function App() {

  return (
      <BrowserRouter>
          <Routes>
              <Route path="/" element={<Navigate to={"/home"} replace />} />
              <Route element={<AppLayout />}>
                  <Route path="/home" element={<WorkspaceHomePage />} />
                  <Route path="/projects/:projectId" element={<ProjectDetailPage />} />
                  <Route path="/projects/:projectId/plans/:planId" element={<PlanDetailPage />} />
              </Route>
              <Route path="*" element={<NotFoundPage />} />
          </Routes>
      </BrowserRouter>
  );
}

export default App
