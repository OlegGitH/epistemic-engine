"use client";

import { FormEvent, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import "./review.css";
import {
  ComparisonSection,
  DecisionReport,
  EmptyRun,
  EvidenceSection,
  HistorySection,
  MachineCertificate,
  ProgressGuide,
  RunHeader,
  RunJumpNavigation,
  RunOverview,
  VerificationSection,
} from "./components";
import {
  Certificate,
  CertificateReport,
  Graph,
  Verification,
  normalizeCertificate,
  normalizeCertificateReport,
  normalizeGraph,
  toFlow,
} from "./model";

const API = process.env.NEXT_PUBLIC_CONTROL_PLANE_URL ?? "http://localhost:8080";

export default function RunPage() {
  return (
    <Suspense fallback={<main className="runPage"><p className="runLoading">Loading run review…</p></main>}>
      <RunDebugger />
    </Suspense>
  );
}

function RunDebugger() {
  const searchParams = useSearchParams();
  const requestedRun = searchParams.get("run") ?? "";
  const live = searchParams.get("live") === "1";

  const [runId, setRunId] = useState(requestedRun);
  const [graph, setGraph] = useState<Graph | null>(null);
  const [selectedClaimId, setSelectedClaimId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [verificationApprovals, setVerificationApprovals] = useState<Record<string, boolean>>({});
  const [humanApproved, setHumanApproved] = useState(false);
  const [certificate, setCertificate] = useState<Certificate | null>(null);
  const [certificateReport, setCertificateReport] = useState<CertificateReport | null>(null);
  const [comparisonRunId, setComparisonRunId] = useState("");
  const [comparison, setComparison] = useState<Graph | null>(null);

  const loadRun = useCallback(async (id: string) => {
    const normalizedId = id.trim();
    if (!normalizedId) return;
    setLoading(true);
    setError("");
    try {
      const next = normalizeGraph(await request<Graph>(`/v1/runs/${normalizedId}/graph`));
      setGraph(next);
      setRunId(normalizedId);
      setSelectedClaimId((current) => next.claims.some((claim) => claim.id === current) ? current : next.claims[0]?.id ?? null);

      const [certificateResult, reportResult] = await Promise.allSettled([
        request<Certificate>(`/v1/decisions/${next.decision.id}/certificate`),
        request<CertificateReport>(`/v1/decisions/${next.decision.id}/certificate/report`),
      ]);
      if (certificateResult.status === "fulfilled") {
        const storedCertificate = normalizeCertificate(certificateResult.value);
        setCertificate(storedCertificate);
        setHumanApproved(storedCertificate.human_approved);
      } else {
        setCertificate(null);
        setHumanApproved(false);
      }
      setCertificateReport(
        reportResult.status === "fulfilled" ? normalizeCertificateReport(reportResult.value) : null,
      );
      window.history.replaceState(null, "", `/run?run=${encodeURIComponent(normalizedId)}${live ? "&live=1" : ""}`);
    } catch (reason) {
      setGraph(null);
      setCertificate(null);
      setCertificateReport(null);
      setError(errorMessage(reason));
    } finally {
      setLoading(false);
    }
  }, [live]);

  useEffect(() => {
    if (requestedRun) void loadRun(requestedRun);
  }, [requestedRun, loadRun]);

  useEffect(() => {
    if (!graph || !live) return;
    const stream = new EventSource(`${API}/v1/runs/${graph.run.id}/events/stream`);
    const refresh = () => void loadRun(graph.run.id);
    const eventTypes = [
      "run.event.appended",
      "analysis.completed",
      "verification.plan.created",
      "verification.completed",
      "decision.evaluated",
    ];
    eventTypes.forEach((type) => stream.addEventListener(type, refresh));
    return () => stream.close();
  }, [graph?.run.id, live, loadRun]); // graph identity intentionally reduced to run ID

  const flow = useMemo(() => toFlow(graph), [graph]);
  const selectedClaim = graph?.claims.find((claim) => claim.id === selectedClaimId) ?? null;
  const selectedEvidence = graph?.evidence.filter((item) => (selectedClaim?.evidence_ids ?? []).includes(item.id)) ?? [];

  async function submitRun(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await loadRun(runId);
  }

  async function postAndRefresh(path: string, body?: unknown) {
    if (!graph) return;
    setLoading(true);
    setError("");
    try {
      await request(path, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: body === undefined ? undefined : JSON.stringify(body),
      });
      await loadRun(graph.run.id);
    } catch (reason) {
      setError(errorMessage(reason));
      setLoading(false);
    }
  }

  async function analyze() {
    if (graph) await postAndRefresh(`/v1/runs/${graph.run.id}/analyze`);
  }

  async function planVerification() {
    if (graph) await postAndRefresh(`/v1/decisions/${graph.decision.id}/verification-plan`);
  }

  async function recordVerification(verification: Verification, outcome: "passed" | "failed") {
    await postAndRefresh(`/v1/verifications/${verification.id}/execute`, {
      environment: "sandbox",
      outcome,
      approved: verificationApprovals[verification.id] === true,
      approved_by: "workspace-reviewer",
      artifact: {
        runner: "reviewed-recorded-artifact",
        outcome,
        recorded_at: new Date().toISOString(),
        check: verification.check,
      },
    });
  }

  async function evaluate() {
    if (!graph) return;
    await postAndRefresh(`/v1/decisions/${graph.decision.id}/evaluate`, { human_approved: humanApproved });
  }

  async function compareRuns() {
    if (!comparisonRunId.trim()) return;
    setError("");
    try {
      setComparison(normalizeGraph(await request<Graph>(`/v1/runs/${comparisonRunId.trim()}/graph`)));
    } catch (reason) {
      setError(errorMessage(reason));
    }
  }

  return (
    <main className="runPage">
      <RunHeader runId={runId} loading={loading} onRunIdChange={setRunId} onSubmit={submitRun} />
      {!graph ? <EmptyRun error={error} /> : (
        <div className="runContent">
          <RunOverview graph={graph} report={certificateReport} />
          {error && <p className="runError" role="alert">{error}</p>}
          <ProgressGuide
            graph={graph}
            certificate={certificate}
            approved={humanApproved}
            loading={loading}
            onApprovedChange={setHumanApproved}
            onAnalyze={analyze}
            onPlan={planVerification}
            onEvaluate={evaluate}
          />
          {certificateReport && <DecisionReport report={certificateReport} onExport={() => downloadText(certificateReport)} />}
          <RunJumpNavigation />
          <EvidenceSection
            graph={graph}
            flow={flow}
            selectedClaim={selectedClaim}
            selectedEvidence={selectedEvidence}
            onSelectClaim={setSelectedClaimId}
          />
          <VerificationSection
            verifications={graph.verifications}
            approvals={verificationApprovals}
            loading={loading}
            onApprovalChange={(id, approved) => setVerificationApprovals((current) => ({ ...current, [id]: approved }))}
            onRecord={recordVerification}
          />
          <HistorySection graph={graph} />
          {certificate && <MachineCertificate certificate={certificate} onExport={() => downloadJson(certificate)} />}
          <ComparisonSection value={comparisonRunId} comparison={comparison} onChange={setComparisonRunId} onCompare={compareRuns} />
        </div>
      )}
    </main>
  );
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API}${path}`, options);
  const text = await response.text();
  const body = text ? JSON.parse(text) : {};
  if (!response.ok) throw new Error(body.error ?? `Request failed with HTTP ${response.status}`);
  return body as T;
}

function downloadJson(certificate: Certificate) {
  downloadFile(
    `decision-certificate-${certificate.decision_id}.json`,
    JSON.stringify(certificate, null, 2),
    "application/json",
  );
}

function downloadText(report: CertificateReport) {
  downloadFile(`decision-report-${report.decision_id}.md`, report.markdown, "text/markdown");
}

function downloadFile(filename: string, content: string, type: string) {
  const url = URL.createObjectURL(new Blob([content], { type }));
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

function errorMessage(reason: unknown) {
  return reason instanceof Error ? reason.message : "The request could not be completed.";
}
