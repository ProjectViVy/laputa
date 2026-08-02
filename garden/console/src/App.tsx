import { Navigate, Route, Routes } from "react-router-dom";
import Shell from "./components/Shell";
import Overview from "./pages/Overview";
import GovernanceMap from "./pages/GovernanceMap";
import Operations from "./pages/Operations";
import RecallTrace from "./pages/RecallTrace";
import ArchitectureLibrary from "./pages/ArchitectureLibrary";
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
        <Route path="/materials" element={<Placeholder titleKey="nav.materials" />} />
        <Route path="/reports" element={<Placeholder titleKey="nav.reports" />} />
        <Route path="/settings" element={<Placeholder titleKey="nav.settings" />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
