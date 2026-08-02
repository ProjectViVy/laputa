import { Outlet } from "react-router-dom";
import Sidebar from "./Sidebar";
import TopBar from "./TopBar";
import SafetyBanner from "./SafetyBanner";

export default function Shell() {
  return (
    <div className="shell">
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
