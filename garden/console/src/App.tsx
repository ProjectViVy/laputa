import { Navigate, Route, Routes } from "react-router-dom";
import Shell from "./components/Shell";
import Overview from "./pages/Overview";
import GovernanceMap from "./pages/GovernanceMap";
import Operations from "./pages/Operations";
import RecallTrace from "./pages/RecallTrace";
import ArchitectureLibrary from "./pages/ArchitectureLibrary";
import MaterialsPage from "./pages/MaterialsPage";
import ReportsPage from "./pages/ReportsPage";
import ModulesPage from "./pages/ModulesPage";
import EvolutionPage from "./pages/EvolutionPage";
import MailboxPage from "./pages/MailboxPage";
import Placeholder from "./pages/Placeholder";

export default function App() {
  return (
    <Routes>
      <Route element={<Shell />}>
        <Route index element={<Overview />} />
        <Route path="/governance" element={<GovernanceMap />} />
        <Route path="/operations" element={<Operations />} />
        <Route path="/trace" element={<RecallTrace />} />
        <Route path="/library" element={<ArchitectureLibrary />} />
        <Route path="/work" element={<Placeholder titleKey="nav.work" />} />
        <Route path="/materials" element={<MaterialsPage />} />
        <Route path="/reports" element={<ReportsPage />} />
        <Route path="/modules" element={<ModulesPage />} />
        <Route path="/evolution" element={<EvolutionPage />} />
        <Route path="/mailbox" element={<MailboxPage />} />
        <Route path="/settings" element={<Placeholder titleKey="nav.settings" />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
