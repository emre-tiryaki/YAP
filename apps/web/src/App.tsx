import { BrowserRouter, Routes, Route } from "react-router"
import WorkspaceHomePage from "./pages/WorkspaceHomePage"

function App() {

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/home" element={<WorkspaceHomePage />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
