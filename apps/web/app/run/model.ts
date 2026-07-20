import { Edge, MarkerType, Node } from "@xyflow/react";

export type ClaimState =
  | "supported"
  | "externally_verified"
  | "contradicted"
  | "rejected"
  | "verification_pending"
  | "unsupported"
  | "partially_supported"
  | "stale";

export type Claim = {
  id: string;
  statement: string;
  scope: string;
  critical: boolean;
  importance: string;
  state: ClaimState;
  justification: string;
  evidence_ids: string[] | null;
  required_evidence_types: string[] | null;
  support: {
    value: number;
    semantics: string;
    evidence_coverage: number;
    freshness: number;
    source_independence: number;
    scope_match: number;
    direct_verification_strength: number;
    contradiction_burden: number;
  };
};

export type Evidence = {
  id: string;
  kind: string;
  source: string;
  summary: string;
  content_hash: string;
};

export type Verification = {
  id: string;
  claim_id: string;
  kind: string;
  check: string;
  status: string;
  outcome?: string;
  environment?: string;
  requires_approval: boolean;
  approved: boolean;
  artifact_hash?: string;
};

export type Graph = {
  run: {
    id: string;
    title: string;
    recommendation: string;
    status: string;
    events: Array<{
      id: string;
      sequence: number;
      type: string;
      source: string;
      occurred_at: string;
    }>;
  };
  decision: {
    id: string;
    verdict?: string;
    action_allowed: boolean;
    policy_version: string;
  };
  claims: Claim[];
  evidence: Evidence[];
  relations: Array<{ id: string; from_id: string; to_id: string; type: string }>;
  unknowns: Array<{ id: string; question: string; critical: boolean; resolved: boolean }>;
  verifications: Verification[];
};

export type Certificate = {
  decision_id: string;
  run_id: string;
  verdict: string;
  action_allowed: boolean;
  human_approved: boolean;
  policy_version: string;
  conditions: string[];
  artifact_hashes: string[];
  issued_at: string;
  proof: { algorithm: string; digest: string };
};

export type CertificateReport = {
  decision_id: string;
  run_id: string;
  decision: string;
  headline: string;
  summary: string;
  recommendation: string;
  verdict: string;
  action_allowed: boolean;
  human_approval_required: boolean;
  human_approval_granted: boolean;
  policy_version: string;
  issued_at: string;
  counts: {
    critical_claims: number;
    supported_claims: number;
    contradicted_claims: number;
    open_claims: number;
    verification_checks: number;
    passed_verifications: number;
    evidence_artifacts: number;
  };
  critical_claims: Array<{
    statement: string;
    state: ClaimState;
    assessment: string;
    support_percent: number;
    evidence_count: number;
  }>;
  conditions: string[];
  proof: { algorithm: string; digest: string };
  markdown: string;
};

export type FlowModel = { nodes: Node[]; edges: Edge[] };

export const claimTones: Record<ClaimState, string> = {
  supported: "#42d392",
  externally_verified: "#42d392",
  contradicted: "#ff6b6b",
  rejected: "#ff6b6b",
  verification_pending: "#f6c85f",
  unsupported: "#8793a8",
  partially_supported: "#f6c85f",
  stale: "#8793a8",
};

export function normalizeGraph(graph: Graph): Graph {
  return {
    ...graph,
    run: { ...graph.run, events: graph.run.events ?? [] },
    claims: graph.claims ?? [],
    evidence: graph.evidence ?? [],
    relations: graph.relations ?? [],
    unknowns: graph.unknowns ?? [],
    verifications: graph.verifications ?? [],
  };
}

export function normalizeCertificate(certificate: Certificate): Certificate {
  return {
    ...certificate,
    conditions: certificate.conditions ?? [],
    artifact_hashes: certificate.artifact_hashes ?? [],
  };
}

export function normalizeCertificateReport(report: CertificateReport): CertificateReport {
  return {
    ...report,
    critical_claims: report.critical_claims ?? [],
    conditions: report.conditions ?? [],
  };
}

export function criticalClaimsOpen(graph: Graph): number {
  return graph.claims.filter(
    (claim) => claim.critical && !["supported", "externally_verified"].includes(claim.state),
  ).length;
}

export function humanize(value: string): string {
  return value.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

export function percent(value: number | undefined): string {
  return `${Math.round((value ?? 0) * 100)}%`;
}

export function toFlow(graph: Graph | null): FlowModel {
  if (!graph) return { nodes: [], edges: [] };

  const evidenceNodes: Node[] = graph.evidence.map((item, index) => ({
    id: item.id,
    position: { x: 0, y: index * 100 },
    data: { label: `${humanize(item.kind)}\n${item.source}`, kind: "evidence" },
    className: "evidenceNode",
  }));
  const claimNodes: Node[] = graph.claims.map((item, index) => ({
    id: item.id,
    position: { x: 340, y: index * 130 },
    data: { label: item.statement, kind: "claim" },
    style: { borderColor: claimTones[item.state] ?? "#8793a8" },
    className: "claimNode",
  }));
  const verificationNodes: Node[] = graph.verifications.map((item, index) => ({
    id: item.id,
    position: { x: 700, y: index * 115 },
    data: { label: `${humanize(item.kind)}\n${humanize(item.outcome ?? item.status)}`, kind: "verification" },
    className: "verificationNode",
  }));
  const decisionNode: Node = {
    id: graph.decision.id,
    position: { x: 1040, y: Math.max(50, (graph.claims.length - 1) * 65) },
    data: { label: humanize(graph.decision.verdict ?? "Deployment decision"), kind: "decision" },
    className: "decisionNode",
  };

  const nodeIds = new Set(
    [...evidenceNodes, ...claimNodes, ...verificationNodes, decisionNode].map((node) => node.id),
  );
  const relationEdges: Edge[] = graph.relations
    .filter((item) => nodeIds.has(item.from_id) && nodeIds.has(item.to_id))
    .map((item) => ({
      id: item.id,
      source: item.from_id,
      target: item.to_id,
      label: item.type,
      markerEnd: { type: MarkerType.ArrowClosed },
      style: { stroke: item.type === "contradicts" ? "#ff6b6b" : "#6686b5" },
    }));
  const claimEdges: Edge[] = graph.claims.map((item) => ({
    id: `decision-${item.id}`,
    source: item.id,
    target: graph.decision.id,
    animated: item.critical,
    style: { stroke: item.critical ? "#b996f7" : "#46536a" },
  }));

  return {
    nodes: [...evidenceNodes, ...claimNodes, ...verificationNodes, decisionNode],
    edges: [...relationEdges, ...claimEdges],
  };
}
