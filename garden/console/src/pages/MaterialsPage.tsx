import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useApi } from "../api/hooks";
import type { CardsResponse, CollectionsResponse, EvidenceResponse, MemoryCard } from "../api/types";
import PageHeader from "../components/PageHeader";

export default function MaterialsPage() {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [activeQuery, setActiveQuery] = useState<string | null>(null);
  const [collection, setCollection] = useState<string | null>(null);
  const [selectedCard, setSelectedCard] = useState<MemoryCard | null>(null);

  const { data: collections } = useApi<CollectionsResponse>("/v2/materials/collections");

  const cardsPath = activeQuery
    ? `/v2/materials/cards?query=${encodeURIComponent(activeQuery)}${collection ? `&collection=${encodeURIComponent(collection)}` : ""}&limit=20`
    : null;
  const { data: cards, loading: cardsLoading, error: cardsError } = useApi<CardsResponse>(cardsPath);

  const evidencePath = selectedCard
    ? `/v2/materials/cards/${encodeURIComponent(selectedCard.id)}/evidence`
    : null;
  const { data: evidence, loading: evidenceLoading } = useApi<EvidenceResponse>(evidencePath);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const v = query.trim();
    if (v) {
      setActiveQuery(v);
      setSelectedCard(null);
    }
  };

  return (
    <div className="page">
      <PageHeader title={t("materials.title")} lede={t("materials.lede")} />

      <form className="materials-search-row" onSubmit={submit}>
        <input
          className="materials-input"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t("materials.searchPlaceholder")}
        />
        <button className="btn" type="submit">{t("materials.search")}</button>
      </form>

      <div className="materials-layout">
        <aside className="materials-sidebar">
          <h3 className="materials-sidebar-title">{t("materials.collections")}</h3>
          <button
            className={`materials-collection-item${!collection ? " active" : ""}`}
            onClick={() => setCollection(null)}
          >
            {t("materials.allCollections")}
          </button>
          {collections?.collections.map((c) => (
            <button
              key={c.name}
              className={`materials-collection-item${collection === c.name ? " active" : ""}`}
              onClick={() => setCollection(c.name)}
            >
              <span>{c.name}</span>
              <span className="materials-count">{c.count}</span>
            </button>
          ))}
          {collections && collections.collections.length === 0 && (
            <p className="materials-empty">{t("materials.noCollections")}</p>
          )}
        </aside>

        <section className="materials-cards">
          {cardsLoading && <p className="materials-empty">{t("common.loading")}…</p>}
          {cardsError && <p className="materials-empty error">{cardsError}</p>}
          {!cardsLoading && cards && cards.cards.length === 0 && (
            <p className="materials-empty">{t("materials.noResults")}</p>
          )}
          {cards?.cards.map((card) => (
            <div
              key={card.id}
              className={`materials-card-item${selectedCard?.id === card.id ? " selected" : ""}`}
              onClick={() => setSelectedCard(card)}
            >
              <div className="materials-card-head">
                <span className="materials-card-title">{card.title}</span>
                <span className={`materials-badge materials-badge--${card.status}`}>{card.status}</span>
              </div>
              <p className="materials-card-summary">{card.summary}</p>
              <div className="materials-card-meta">
                {card.collection && <span className="materials-tag">{card.collection}</span>}
                {card.tags.map((tag) => (
                  <span key={tag} className="materials-tag">{tag}</span>
                ))}
                <span className="materials-heat">{t("materials.heat")}: {card.heat_score.toFixed(2)}</span>
              </div>
            </div>
          ))}
        </section>

        <aside className="materials-inspector">
          {!selectedCard && (
            <p className="materials-empty">{t("materials.selectCard")}</p>
          )}
          {selectedCard && (
            <>
              <h3 className="materials-inspector-title">{t("materials.evidence")}</h3>
              <p className="materials-inspector-card">{selectedCard.title}</p>
              {evidenceLoading && <p className="materials-empty">{t("common.loading")}…</p>}
              {!evidenceLoading && evidence && evidence.fragments.length === 0 && (
                <p className="materials-empty">{t("materials.noEvidence")}</p>
              )}
              {evidence?.fragments.map((frag, i) => (
                <div key={i} className="materials-fragment">
                  <p className="materials-fragment-excerpt">{frag.excerpt}</p>
                  <div className="materials-fragment-meta">
                    <span>{t("materials.validity")}: <strong>{frag.validity}</strong></span>
                    {frag.source_uri && <span>{t("materials.sourceUri")}: {frag.source_uri}</span>}
                    <span>{t("materials.materialRef")}: {frag.material_ref}</span>
                    <span className="mono">{t("materials.hash")}: {frag.content_hash.slice(0, 12)}…</span>
                  </div>
                </div>
              ))}
            </>
          )}
        </aside>
      </div>
    </div>
  );
}
