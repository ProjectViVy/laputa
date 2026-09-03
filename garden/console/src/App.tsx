import { Navigate, Route, Routes } from "react-router-dom";
import Shell from "./components/Shell";
import Overview from "./pages/Overview";
import PersonaPage from "./modules/persona/PersonaPage";
import MemoryPage from "./modules/memory/MemoryPage";
import ChatApprovalPage from "./modules/chat-approval/ChatApprovalPage";
import Operations from "./pages/Operations";
import RecallTrace from "./pages/RecallTrace";
import ArchitectureLibrary from "./pages/ArchitectureLibrary";
import MaterialsPage from "./pages/MaterialsPage";
import ReportsPage from "./pages/ReportsPage";
import ModulesPage from "./pages/ModulesPage";
import EvolutionPage from "./pages/EvolutionPage";
import SettingsPage from "./pages/SettingsPage";

export default function App() {
  return (
    <Routes>
      <Route element={<Shell />}>
        <Route index element={<Overview />} />
        <Route path="/persona" element={<PersonaPage />} />
        <Route path="/memory" element={<MemoryPage />} />
        <Route path="/chat-approval" element={<ChatApprovalPage />} />
        <Route path="/operations" element={<Operations />} />
        <Route path="/trace" element={<RecallTrace />} />
        <Route path="/library" element={<ArchitectureLibrary />} />
        <Route path="/materials" element={<MaterialsPage />} />
        <Route path="/reports" element={<ReportsPage />} />
        <Route path="/modules" element={<ModulesPage />} />
        <Route path="/evolution" element={<EvolutionPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
