import { NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";

interface Item {
  to: string;
  key: string;
  icon: string;
  soon?: boolean;
}

const ITEMS: Item[] = [
  { to: "/", key: "overview", icon: "M3 12h4l2 5 4-12 2 7h6" },
  { to: "/governance", key: "governance", icon: "M12 3v6m0 6v6M5 8l7 4 7-4M5 16l7-4 7 4" },
  { to: "/work", key: "work", icon: "M4 6h16M4 12h10M4 18h7", soon: true },
  { to: "/materials", key: "materials", icon: "M4 5h10l6 6v8H4zM14 5v6h6", soon: true },
  { to: "/trace", key: "trace", icon: "M12 4v4m0 4v4m0 4v0M6 8h.01M18 16h.01M6 8a6 6 0 0112 8" },
  { to: "/reports", key: "reports", icon: "M6 3h9l4 4v14H6zM9 12h6M9 16h6", soon: true },
  { to: "/operations", key: "operations", icon: "M4 7h16M4 12h16M4 17h16M8 5v4M15 10v4M10 15v4" },
  { to: "/library", key: "library", icon: "M5 4h6v16H5zM13 4h6v16h-6z" },
  { to: "/settings", key: "settings", icon: "M12 8a4 4 0 100 8 4 4 0 000-8zM12 2v3m0 14v3M2 12h3m14 0h3", soon: true },
];

function Icon({ d }: { d: string }) {
  return (
    <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
      <path d={d} />
    </svg>
  );
}

export default function Sidebar() {
  const { t } = useTranslation();
  return (
    <aside className="sidebar">
      <div className="brand">
        <div className="brand-mark">M</div>
        <div className="brand-text">
          <span className="brand-name">{t("app.name")}</span>
          <span className="brand-sub">{t("app.subtitle")}</span>
        </div>
      </div>
      <nav className="nav">
        {ITEMS.map((item) => (
          <NavLink
            key={item.key}
            to={item.to}
            end={item.to === "/"}
            className={({ isActive }) => `nav-item${isActive ? " active" : ""}`}
          >
            <span className="nav-icon">
              <Icon d={item.icon} />
            </span>
            <span className="nav-label">{t(`nav.${item.key}`)}</span>
            {item.soon && <span className="nav-soon" title={t("common.notImplemented")} />}
          </NavLink>
        ))}
      </nav>
      <div className="sidebar-foot mono">{t("app.contract")}</div>
    </aside>
  );
}
