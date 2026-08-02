import { Outlet } from "react-router-dom";
import Sidebar from "./Sidebar";
import TopBar from "./TopBar";
import SafetyBanner from "./SafetyBanner";
import { usePreview } from "../lib/usePreview";

export default function Shell() {
  const { active } = usePreview();
  return (
    <div className={`shell${active ? " preview-mode" : ""}`}>
      <Sidebar />
      <div className="shell-main">
        <TopBar />
        <SafetyBanner />
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
