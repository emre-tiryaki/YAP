import { BrowserRouter, Routes, Route, Navigate } from "react-router"
import WorkspaceHomePage from "./pages/WorkspaceHomePage"
import NotFoundPage from "./pages/NotFoundPage";

function App() {

  return (
      <BrowserRouter>
          <Routes>
              <Route path="/" element={<Navigate to={"/home"} replace />} />
              <Route path="/home" element={<WorkspaceHomePage />} />
              <Route path="*" element={<NotFoundPage />} />
          </Routes>
      </BrowserRouter>
  );
}

export default App
