import PageHeader from "../../components/PageHeader";

export default function ChatApprovalPage() {
  return (
    <div className="page">
      <PageHeader title="Chat Approval" lede="Dangerous runtime operations are reviewed here, separately from Persona content and EvoMap artifacts." />
      <section className="panel panel-pad approval-empty">
        <span className="chip">no live request feed</span>
        <h2 className="section-title">Approval service is not exposed</h2>
        <p className="source-note">This Garden build does not publish a Chat Approval endpoint. No request or action is fabricated in the Console.</p>
      </section>
    </div>
  );
}
