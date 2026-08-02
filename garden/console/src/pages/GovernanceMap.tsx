import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import type { OverviewResponse } from "../api/types";
import PageHeader from "../components/PageHeader";
import Inspector from "../components/Inspector";
import StatusDot from "../components/StatusDot";
import { FLOW_DESCRIPTIONS, GOV_NODES, type GovNode } from "../data/governance";
import { useBi } from "../lib/bi";
import { statusTone, TONE_CLASS } from "../lib/status";

const FLOWS = ["context", "ingestion", "recall", "change"] as const;
type Flow = (typeof FLOWS)[number];

interface Edge {
  id: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  compat?: boolean;
  flows: Flow[];
}

const EDGES: Edge[] = [
  { id: "host-garden", x1: 50, y1: 12.5, x2: 50, y2: 37.5, flows: ["context", "ingestion", "recall"] },
  { id: "garden-laputa", x1: 50, y1: 37.5, x2: 16.7, y2: 62.5, flows: ["context", "ingestion", "change"] },
  { id: "garden-mentle", x1: 50, y1: 37.5, x2: 50, y2: 62.5, flows: ["context", "ingestion", "recall"] },
  { id: "garden-legacy", x1: 50, y1: 37.5, x2: 83.3, y2: 62.5, compat: true, flows: [] },
  { id: "laputa-ctx", x1: 16.7, y1: 62.5, x2: 50, y2: 87.5, flows: ["context", "recall"] },
  { id: "mentle-ctx", x1: 50, y1: 62.5, x2: 50, y2: 87.5, flows: ["context", "recall"] },
];

const COL_X = [16.7, 50, 83.3];
const ROW_Y = [12.5, 37.5, 62.5, 87.5];

export default function GovernanceMap() {
  const { t } = useTranslation();
  const bi = useBi();
  const { data } = useApi<OverviewResponse>("/v2/admin/overview", { poll: 8000 });
  const [flow, setFlow] = useState<Flow>("context");
  const [selected, setSelected] = useState<GovNode | null>(null);

  const components = data?.components ?? {};
  const spool = data?.spool_pending ?? 0;

  const runtimeFor = (node: GovNode): string | undefined => {
    if (node.component) return components[node.component];
    if (node.compat) return "compat";
    return undefined;
  };

  const dataLayer = (node: GovNode): string => {
    if (node.id === "garden" || node.id === "mentle") return `${spool} pending`;
    return "—";
  };

  return (
    <div className="page">
      <PageHeader title={t("governance.title")} lede={t("governance.lede")} />

      <div className="flow-tabs">
        {FLOWS.map((f) => (
          <button
            key={f}
            className={`flow-tab mono${flow === f ? " active" : ""}`}
            onClick={() => setFlow(f)}
          >
            {t(`governance.flow.${f}`)}
          </button>
        ))}
      </div>
      <p className="flow-desc mono">{bi(FLOW_DESCRIPTIONS[flow])}</p>

      <div className="map-wrap panel reveal">
        <svg className="map-svg" viewBox="0 0 100 100" preserveAspectRatio="none">
          {EDGES.map((e) => {
            const active = e.flows.includes(flow);
            return (
              <line
                key={e.id}
                x1={e.x1}
                y1={e.y1}
                x2={e.x2}
                y2={e.y2}
                className={`edge${active ? " edge-active" : ""}${e.compat ? " edge-compat" : ""}`}
                vectorEffect="non-scaling-stroke"
              />
            );
          })}
        </svg>

        {GOV_NODES.map((node) => {
          const rt = runtimeFor(node);
          const rtTone = node.compat ? "degraded" : statusTone(rt);
          return (
            <button
              key={node.id}
              className={`node${node.compat ? " node-compat" : ""}${selected?.id === node.id ? " node-selected" : ""}`}
              style={{ left: `${COL_X[node.col]}%`, top: `${ROW_Y[node.tier]}%` }}
              onClick={() => setSelected(node)}
            >
              <div className="node-head">
                <span className="node-name">{node.name}</span>
                <StatusDot tone={rtTone} size={7} pulse={!node.compat} />
              </div>
              <div className="node-layers">
                <Layer k={t("governance.layers.runtime")} v={rt ?? t("status.acceptedDesign")} tone={rtTone} />
                <Layer k={t("governance.layers.data")} v={dataLayer(node)} />
                <Layer k={t("governance.layers.governance")} v={bi(node.governanceNote)} dim />
                <Layer k={t("governance.layers.roadmap")} v={bi(node.roadmapNote)} dim />
              </div>
            </button>
          );
        })}
      </div>

      <Inspector node={selected} runtimeStatus={selected ? runtimeFor(selected) : undefined} onClose={() => setSelected(null)} />
    </div>
  );
}

function Layer({ k, v, tone, dim }: { k: string; v: string; tone?: string; dim?: boolean }) {
  return (
    <div className="node-layer">
      <span className="node-layer-k mono">{k}</span>
      <span className={`node-layer-v mono${tone ? ` ${TONE_CLASS[statusTone(tone)]}` : ""}${dim ? " dim" : ""}`}>{v}</span>
    </div>
  );
}
