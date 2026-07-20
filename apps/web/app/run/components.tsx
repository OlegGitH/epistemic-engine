"use client";

import Link from "next/link";
import { Background, Controls, ReactFlow } from "@xyflow/react";
import {
  Certificate,
  CertificateReport,
  Claim,
  Evidence,
  FlowModel,
  Graph,
  Verification,
  claimTones,
  criticalClaimsOpen,
  humanize,
  percent,
} from "./model";

type RunHeaderProps = {
  runId: string;
  loading: boolean;
  onRunIdChange: (value: string) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
};

export function RunHeader({ runId, loading, onRunIdChange, onSubmit }: RunHeaderProps) {
  return (
    <header className="runTopbar">
      <Link className="runBrand" href="/">
        <span className="runBrandMark">E</span>
        <span>
          <strong>Epistemic</strong>
          <small>Control Center</small>
        </span>
      </Link>
      <nav className="runContext" aria-label="Breadcrumb">
        <Link href="/">Portfolio</Link>
        <span aria-hidden="true">/</span>
        <strong>Run review</strong>
      </nav>
      <form className="runSearch" onSubmit={onSubmit}>
        <label className="srOnly" htmlFor="run-id">Run ID</label>
        <input
          id="run-id"
          placeholder="Paste a run ID"
          value={runId}
          onChange={(event) => onRunIdChange(event.target.value)}
        />
        <button disabled={loading || !runId.trim()}>{loading ? "Loading…" : "Open run"}</button>
      </form>
    </header>
  );
}

export function EmptyRun({ error }: { error: string }) {
  return (
    <section className="runEmptyState">
      <p className="eyebrow">Decision review</p>
      <h1>Understand the evidence before acting.</h1>
      <p>
        Enter a run ID to see the outcome, the reason behind it, unresolved work, and the immutable
        certificate in one review flow.
      </p>
      <ol className="emptyJourney">
        <li><span>1</span><b>Read the decision</b><small>Start with proceed or do not proceed.</small></li>
        <li><span>2</span><b>Review the basis</b><small>Inspect claims, evidence, and contradictions.</small></li>
        <li><span>3</span><b>Resolve or certify</b><small>Complete checks and record the human decision.</small></li>
      </ol>
      {error && <p className="runError" role="alert">{error}</p>}
    </section>
  );
}

export function RunOverview({ graph, report }: { graph: Graph; report: CertificateReport | null }) {
  const allowed = report?.action_allowed ?? graph.decision.action_allowed;
  const decision = report?.decision ?? (graph.decision.verdict ? humanize(graph.decision.verdict) : "Pending");
  return (
    <section className="runOverview">
      <div>
        <p className="runMeta">Run {graph.run.id} · Policy {graph.decision.policy_version}</p>
        <h1>{graph.run.title}</h1>
        <p className="runRecommendation"><span>Proposed action</span>{graph.run.recommendation}</p>
      </div>
      <div className={`runOutcome ${allowed ? "allowed" : "blocked"}`}>
        <small>Current decision</small>
        <strong>{decision}</strong>
        <span>{allowed ? "Action is permitted" : "Action is not permitted"}</span>
      </div>
    </section>
  );
}

type ProgressGuideProps = {
  graph: Graph;
  certificate: Certificate | null;
  approved: boolean;
  loading: boolean;
  onApprovedChange: (approved: boolean) => void;
  onAnalyze: () => void;
  onPlan: () => void;
  onEvaluate: () => void;
};

export function ProgressGuide(props: ProgressGuideProps) {
  const { graph, certificate, approved, loading } = props;
  const openClaims = criticalClaimsOpen(graph);
  const checksComplete = graph.verifications.every((item) => item.status === "completed");
  const steps = [
    { label: "Analyze evidence", detail: graph.claims.length ? `${graph.claims.length} claims identified` : "Not started", done: graph.claims.length > 0 },
    { label: "Resolve evidence gaps", detail: openClaims ? `${openClaims} critical gates open` : "Critical gates resolved", done: graph.claims.length > 0 && openClaims === 0 && checksComplete },
    { label: "Record human decision", detail: certificate?.human_approved ? "Approval recorded" : certificate ? "Approval not granted" : approved ? "Ready to record" : "Waiting for reviewer", done: certificate?.human_approved === true },
    { label: "Issue certificate", detail: certificate ? humanize(certificate.verdict) : "Not issued", done: certificate !== null },
  ];

  return (
    <section className="runProgress" aria-labelledby="progress-title">
      <header>
        <div><p className="eyebrow">Review progress</p><h2 id="progress-title">From evidence to decision</h2></div>
        <span>{steps.filter((step) => step.done).length} of 4 complete</span>
      </header>
      <ol>
        {steps.map((step, index) => {
          const previousDone = index === 0 || steps.slice(0, index).every((item) => item.done);
          const status = step.done ? "complete" : previousDone && !certificate ? "current" : certificate ? "closed" : "upcoming";
          return <li className={status} key={step.label}><i>{step.done ? "✓" : index + 1}</i><div><b>{step.label}</b><small>{step.detail}</small></div></li>;
        })}
      </ol>
      {certificate ? (
        <p className="progressNotice">This run is finalized. Review the report below or start a new run when evidence changes.</p>
      ) : (
        <div className="progressActions">
          {!graph.claims.length && <button onClick={props.onAnalyze} disabled={loading}>Analyze evidence</button>}
          {!!graph.claims.length && openClaims > 0 && !graph.verifications.length && (
            <button onClick={props.onPlan} disabled={loading}>Plan required checks</button>
          )}
          {!!graph.claims.length && (
            <>
              <label>
                <input type="checkbox" checked={approved} onChange={(event) => props.onApprovedChange(event.target.checked)} />
                I reviewed the evidence and approve the proposed action
              </label>
              <button className="primaryAction" onClick={props.onEvaluate} disabled={loading || !approved}>
                Evaluate and issue certificate
              </button>
            </>
          )}
        </div>
      )}
    </section>
  );
}

export function DecisionReport({ report, onExport }: { report: CertificateReport; onExport: () => void }) {
  return (
    <section className={`decisionReport ${report.action_allowed ? "allowed" : "blocked"}`} aria-labelledby="decision-title">
      <header>
        <div>
          <p className="eyebrow">Human decision report</p>
          <h2 id="decision-title">{report.headline}</h2>
        </div>
        <span className="decisionBadge">{report.decision}</span>
      </header>
      <p className="decisionSummary">{report.summary}</p>
      <dl className="decisionFacts">
        <div><dt>Critical claims</dt><dd>{report.counts.supported_claims}/{report.counts.critical_claims} supported</dd></div>
        <div><dt>Contradictions</dt><dd>{report.counts.contradicted_claims || "None"}</dd></div>
        <div><dt>Evidence artifacts</dt><dd>{report.counts.evidence_artifacts}</dd></div>
        <div><dt>Human approval</dt><dd>{report.human_approval_granted ? "Granted" : "Not granted"}</dd></div>
      </dl>
      <div className="decisionBasis">
        <h3>Why the Engine reached this decision</h3>
        {report.critical_claims.map((claim) => (
          <article key={claim.statement}>
            <span className={`claimState ${claim.state}`}>{humanize(claim.state)}</span>
            <div><b>{claim.statement}</b><p>{claim.assessment}</p></div>
            <small>{claim.support_percent}/100 evidence strength · {claim.evidence_count} artifacts</small>
          </article>
        ))}
      </div>
      {!!report.conditions.length && (
        <div className="decisionConditions">
          <h3>Conditions before action</h3>
          <ul>{report.conditions.map((condition) => <li key={condition}>{condition}</li>)}</ul>
        </div>
      )}
      <footer>
        <p><b>Machine-verifiable source:</b> certificate digest <code>{report.proof.digest.slice(0, 18)}…</code></p>
        <button onClick={onExport}>Download human report</button>
      </footer>
    </section>
  );
}

export function RunJumpNavigation() {
  return (
    <nav className="runJumpNav" aria-label="Run review sections">
      <a href="#evidence">Evidence</a>
      <a href="#checks">Checks</a>
      <a href="#history">History and gaps</a>
      <a href="#machine-certificate">Machine certificate</a>
    </nav>
  );
}

type EvidenceSectionProps = {
  graph: Graph;
  flow: FlowModel;
  selectedClaim: Claim | null;
  selectedEvidence: Evidence[];
  onSelectClaim: (claimId: string) => void;
};

export function EvidenceSection(props: EvidenceSectionProps) {
  const { graph, flow, selectedClaim, selectedEvidence, onSelectClaim } = props;
  return (
    <section className="runSection" id="evidence">
      <SectionHeading eyebrow="Decision basis" title="Evidence and critical claims" detail="Start with the readable claim list. Use the map when you need provenance and relationships." />
      <div className="runMetrics">
        <Metric value={graph.claims.length} label="Claims analyzed" />
        <Metric value={graph.evidence.length} label="Evidence artifacts" />
        <Metric value={criticalClaimsOpen(graph)} label="Critical gates open" tone={criticalClaimsOpen(graph) ? "warn" : "good"} />
        <Metric value={graph.unknowns.filter((item) => !item.resolved).length} label="Unresolved unknowns" />
      </div>
      <div className="claimNavigator" aria-label="Claims">
        {graph.claims.map((claim) => (
          <button className={selectedClaim?.id === claim.id ? "selected" : ""} key={claim.id} onClick={() => onSelectClaim(claim.id)} aria-pressed={selectedClaim?.id === claim.id}>
            <span className={`claimState ${claim.state}`}>{humanize(claim.state)}</span>
            <b>{claim.statement}</b>
            <small>{Math.round(claim.support.value * 100)}/100 evidence strength</small>
          </button>
        ))}
      </div>
      <div className="evidenceWorkspace">
        <div className="evidenceMap">
          <header><h3>Evidence relationship map</h3><p>Artifacts → claims → decision</p></header>
          <ReactFlow nodes={flow.nodes} edges={flow.edges} fitView onNodeClick={(_, node) => node.data.kind === "claim" && onSelectClaim(node.id)}>
            <Background color="#273348" gap={24} />
            <Controls />
          </ReactFlow>
        </div>
        <ClaimInspector claim={selectedClaim} evidence={selectedEvidence} />
      </div>
    </section>
  );
}

function ClaimInspector({ claim, evidence }: { claim: Claim | null; evidence: Evidence[] }) {
  if (!claim) return <aside className="claimInspector"><p className="muted">Select a claim to inspect its evidence.</p></aside>;
  return (
    <aside className="claimInspector">
      <header><span className={`claimState ${claim.state}`}>{humanize(claim.state)}</span><small>{claim.critical ? "Critical claim" : "Supporting claim"}</small></header>
      <h3>{claim.statement}</h3>
      <p>{claim.justification}</p>
      <div className="evidenceScore">
        <strong>{Math.round(claim.support.value * 100)}</strong><span>/100 evidence strength</span>
        <small>Evidence strength is an explainable support score, not probability.</small>
      </div>
      <dl className="scoreBreakdown">
        <div><dt>Coverage</dt><dd>{percent(claim.support.evidence_coverage)}</dd></div>
        <div><dt>Freshness</dt><dd>{percent(claim.support.freshness)}</dd></div>
        <div><dt>Independence</dt><dd>{percent(claim.support.source_independence)}</dd></div>
        <div><dt>Direct verification</dt><dd>{percent(claim.support.direct_verification_strength)}</dd></div>
      </dl>
      <h4>Required evidence</h4>
      <p>{(claim.required_evidence_types ?? []).map(humanize).join(", ") || "No specific evidence type required"}</p>
      <h4>Bound evidence</h4>
      {evidence.length ? evidence.map((item) => (
        <article className="boundEvidence" key={item.id}>
          <div><b>{humanize(item.kind)}</b><span>{item.source}</span></div>
          <p>{item.summary}</p>
          <code>{item.content_hash.slice(0, 20)}…</code>
        </article>
      )) : <p className="muted">No evidence is bound to this claim.</p>}
    </aside>
  );
}

type VerificationSectionProps = {
  verifications: Verification[];
  approvals: Record<string, boolean>;
  loading: boolean;
  onApprovalChange: (id: string, approved: boolean) => void;
  onRecord: (verification: Verification, outcome: "passed" | "failed") => void;
};

export function VerificationSection(props: VerificationSectionProps) {
  return (
    <section className="runSection" id="checks">
      <SectionHeading eyebrow="Bounded verification" title="Required checks" detail="Checks run only in an approved sandbox and produce content-addressed artifacts." />
      {!props.verifications.length ? (
        <div className="positiveEmpty"><span>✓</span><div><b>No additional checks were required</b><p>The supplied evidence satisfied the policy without executing a verification plan.</p></div></div>
      ) : (
        <div className="verificationList">{props.verifications.map((item) => (
          <article key={item.id}>
            <div><span className={`checkStatus ${item.outcome ?? item.status}`}>{humanize(item.outcome ?? item.status)}</span><h3>{item.check}</h3><p>{humanize(item.kind)} · {item.environment ?? "sandbox"}</p>{item.artifact_hash && <code>{item.artifact_hash}</code>}</div>
            {item.status !== "completed" && <div className="verificationActions"><label><input type="checkbox" checked={props.approvals[item.id] ?? false} onChange={(event) => props.onApprovalChange(item.id, event.target.checked)} />Approve this bounded check</label><button disabled={!props.approvals[item.id] || props.loading} onClick={() => props.onRecord(item, "passed")}>Record pass</button><button className="danger" disabled={!props.approvals[item.id] || props.loading} onClick={() => props.onRecord(item, "failed")}>Record failure</button></div>}
          </article>
        ))}</div>
      )}
    </section>
  );
}

export function HistorySection({ graph }: { graph: Graph }) {
  return (
    <section className="runSection" id="history">
      <SectionHeading eyebrow="Audit trail" title="History and unresolved gaps" detail="Every observation and remaining uncertainty stays visible after the decision." />
      <div className="historyGrid">
        <div className="timelineList"><h3>Run timeline</h3>{graph.run.events.map((event) => <article key={event.id}><i /><div><b>{event.sequence}. {humanize(event.type)}</b><span>{event.source} · {new Date(event.occurred_at).toLocaleString()}</span></div></article>)}</div>
        <div className="unknownList"><h3>Unknowns and gates</h3>{graph.unknowns.length ? graph.unknowns.map((item) => <article key={item.id}><span className={item.resolved ? "resolved" : item.critical ? "critical" : "open"}>{item.resolved ? "Resolved" : item.critical ? "Critical" : "Open"}</span><p>{item.question}</p></article>) : <div className="positiveEmpty compact"><span>✓</span><div><b>No unresolved unknowns</b><p>The decision contains no open knowledge gaps.</p></div></div>}</div>
      </div>
    </section>
  );
}

export function MachineCertificate({ certificate, onExport }: { certificate: Certificate; onExport: () => void }) {
  return (
    <details className="machineCertificate" id="machine-certificate">
      <summary><span><small>Audit and integration details</small><b>Machine certificate and integrity proof</b></span><i>Show details</i></summary>
      <div>
        <dl><div><dt>Verdict</dt><dd>{humanize(certificate.verdict)}</dd></div><div><dt>Policy</dt><dd>{certificate.policy_version}</dd></div><div><dt>Issued</dt><dd>{new Date(certificate.issued_at).toLocaleString()}</dd></div><div><dt>Artifacts</dt><dd>{certificate.artifact_hashes.length}</dd></div></dl>
        <div className="certificateProof"><span>{certificate.proof.algorithm}</span><code>{certificate.proof.digest}</code><button onClick={onExport}>Download certificate JSON</button></div>
      </div>
    </details>
  );
}

export function ComparisonSection({ value, comparison, onChange, onCompare }: { value: string; comparison: Graph | null; onChange: (value: string) => void; onCompare: () => void }) {
  return (
    <section className="comparisonSection">
      <div><p className="eyebrow">Compare decisions</p><h2>What changed between runs?</h2><p>Load another run to compare verdicts and unresolved critical gates.</p></div>
      <div className="compareControl"><input aria-label="Comparison run ID" placeholder="Paste another run ID" value={value} onChange={(event) => onChange(event.target.value)} /><button disabled={!value.trim()} onClick={onCompare}>Compare</button></div>
      {comparison && <div className="comparisonResult"><b>{humanize(comparison.decision.verdict ?? "Unevaluated")}</b><span>{criticalClaimsOpen(comparison)} critical gates open</span><Link href={`/run?run=${comparison.run.id}`}>Open compared run →</Link></div>}
    </section>
  );
}

function SectionHeading({ eyebrow, title, detail }: { eyebrow: string; title: string; detail: string }) {
  return <header className="runSectionHeading"><div><p className="eyebrow">{eyebrow}</p><h2>{title}</h2></div><p>{detail}</p></header>;
}

function Metric({ value, label, tone = "" }: { value: number; label: string; tone?: string }) {
  return <div className={`runMetric ${tone}`}><strong>{value}</strong><span>{label}</span></div>;
}
