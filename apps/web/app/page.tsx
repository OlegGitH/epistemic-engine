"use client";

import Link from "next/link";
import { FormEvent, ReactNode, useEffect, useMemo, useState } from "react";
import "./portfolio.css";

type Account = { id: string; name: string; slug: string; created_at: string };
type Metrics = { projects: number; connected_projects: number; ai_systems: number; reports: number; valid_certificates: number; attention_items: number; knowledge_coverage_pct: number };
type Knowledge = { claims: number; supported_claims: number; contradicted_claims: number; evidence_artifacts: number; open_unknowns: number; stale_claims: number };
type Project = { id: string; name: string; repository?: string; owner?: string; runs: number; reports: number; ai_systems: number; claims: number; supported_claims: number; evidence_artifacts: number; open_unknowns: number; knowledge_coverage_pct: number; certification_status: Status; connection_status: "active" | "revoked" | "disconnected"; latest_run_id?: string; last_activity_at?: string };
type AISystem = { id: string; project_id: string; project_name: string; name: string; provider: string; model: string; purpose: string; data_classes: string[]; tools: string[]; owner?: string; status: string; certification_status: Status; certificate_digest?: string; last_evaluated_at?: string; last_used_at?: string };
type Certificate = { decision_id: string; run_id: string; project_id: string; project_name: string; ai_system_name?: string; verdict: string; action_allowed: boolean; policy_version: string; issued_at: string; digest: string; source?: string };
type Activity = { id: string; kind: string; title: string; detail: string; status: string; occurred_at: string; run_id?: string };
type Report = { id:string; external_id:string; project_id:string; ai_system_id?:string; tool:string; status:string; exit_code:number; summary:string; repository?:string; commit_sha?:string; branch?:string; workflow?:string; run_url?:string; details?:{run_id?:string}; received_at:string };
type Connection = { id:string; project_id:string; project_name:string; provider:string; repository:string; status:string; token_prefix:string; reports:number; created_at:string; last_seen_at?:string };
type Dashboard = { account: Account; metrics: Metrics; knowledge: Knowledge; projects: Project[]; ai_systems: AISystem[]; connections:Connection[]; reports:Report[]; certificates: Certificate[]; activity: Activity[]; generated_at: string };
type ConnectionSetup = { connection:Connection; token:string; workflow:string };
type Status = "valid" | "blocked" | "pending" | "uncertified";
type Tab = "overview" | "projects" | "ai" | "reports" | "certificates" | "knowledge";

const API = process.env.NEXT_PUBLIC_CONTROL_PLANE_URL ?? "http://localhost:8080";
const CONFIGURED_ACCOUNT = process.env.NEXT_PUBLIC_EPISTEMIC_ACCOUNT_ID ?? "";

export default function DashboardPage() {
  const [accountId, setAccountId] = useState("");
  const [dashboard, setDashboard] = useState<Dashboard | null>(null);
  const [tab, setTab] = useState<Tab>("overview");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [dialog, setDialog] = useState<"workspace" | "project" | "ai" | "connect" | null>(null);
  const [connectionSetup, setConnectionSetup] = useState<ConnectionSetup | null>(null);

  async function request<T>(path: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${API}${path}`, options);
    const body = await response.json();
    if (!response.ok) throw new Error(body.error ?? `HTTP ${response.status}`);
    return body;
  }

  async function load(id = accountId) {
    if (!id) return;
    setLoading(true);
    setError("");
    try {
      const value = await request<Dashboard>(`/v1/accounts/${id}/dashboard`);
      setDashboard(value);
      setAccountId(id);
      localStorage.setItem("epistemic-account-id", id);
    } catch (reason) {
      setDashboard(null);
      setError(message(reason));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const sharedAccount = params.get("account") ?? "";
    const requestedTab = params.get("view");
    if (isTab(requestedTab)) setTab(requestedTab);
    const saved = sharedAccount || CONFIGURED_ACCOUNT || localStorage.getItem("epistemic-account-id") || "";
    if (saved) {
      setAccountId(saved);
      void load(saved);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const syncTabFromHistory = () => {
      const requestedTab = new URLSearchParams(window.location.search).get("view");
      setTab(isTab(requestedTab) ? requestedTab : "overview");
    };
    window.addEventListener("popstate", syncTabFromHistory);
    return () => window.removeEventListener("popstate", syncTabFromHistory);
  }, []);

  function selectTab(nextTab: Tab) {
    setTab(nextTab);
    const url = new URL(window.location.href);
    if (nextTab === "overview") url.searchParams.delete("view");
    else url.searchParams.set("view", nextTab);
    window.history.pushState(null, "", url);
  }

  async function createWorkspace(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setLoading(true);
    try {
      const account = await request<Account>("/v1/accounts", jsonRequest({ name: form.get("name"), slug: form.get("slug") || undefined }));
      setAccountId(account.id);
      localStorage.setItem("epistemic-account-id", account.id);
      setDialog(null);
      await load(account.id);
    } catch (reason) {
      setError(message(reason));
    } finally {
      setLoading(false);
    }
  }

  async function openWorkspace(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const id = String(new FormData(event.currentTarget).get("account_id") ?? "").trim();
    if (!id) return;
    setDialog(null);
    setAccountId(id);
    await load(id);
  }

  async function createProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setLoading(true);
    try {
      await request(`/v1/accounts/${accountId}/projects`, jsonRequest({ name: form.get("name"), repository: form.get("repository"), owner: form.get("owner") }));
      setDialog(null);
      await load();
    } catch (reason) {
      setError(message(reason));
    } finally {
      setLoading(false);
    }
  }

  async function createAISystem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const projectId = String(form.get("project_id"));
    setLoading(true);
    try {
      await request(`/v1/projects/${projectId}/ai-systems`, jsonRequest({
        name: form.get("name"), provider: form.get("provider"), model: form.get("model"), purpose: form.get("purpose"), owner: form.get("owner"),
        data_classes: splitList(form.get("data_classes")), tools: splitList(form.get("tools")),
      }));
      setDialog(null);
      await load();
    } catch (reason) {
      setError(message(reason));
    } finally {
      setLoading(false);
    }
  }

  async function createConnection(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const projectId = String(form.get("project_id"));
    setLoading(true);
    try {
      const setup = await request<ConnectionSetup>(`/v1/projects/${projectId}/connections`, jsonRequest({ provider:"github-actions", repository:form.get("repository"), endpoint:form.get("endpoint") }));
      setConnectionSetup(setup);
      await load();
    } catch (reason) {
      setError(message(reason));
    } finally {
      setLoading(false);
    }
  }

  if (!dashboard) {
    return <main className="shell onboardingShell">
      <BrandBar accountName="No workspace selected" onSwitch={() => setDialog("workspace")} />
      <section className="onboarding">
        <div className="onboardingCopy">
          <p className="eyebrow">Evidence operations</p>
          <h1>One control center for every AI decision.</h1>
          <p>Track project knowledge, registered AI usage, verification health, and immutable decision certificates across your account.</p>
          <div className="onboardingActions"><button className="primary" onClick={() => setDialog("workspace")}>Create workspace</button><Link className="secondaryButton" href="/run">Open run debugger</Link></div>
          {error && <p className="errorBanner">{error}</p>}
        </div>
        <div className="signalPreview" aria-hidden="true">
          <div className="previewHead"><span>Portfolio health</span><StatusPill status="valid" /></div>
          <div className="previewScore"><strong>92</strong><span>evidence coverage</span></div>
          <div className="previewBars"><i style={{width:"92%"}}/><i style={{width:"76%"}}/><i style={{width:"64%"}}/></div>
          <div className="previewRows"><span>AI systems <b>08</b></span><span>Certificates <b>21</b></span><span>Open risks <b className="warnText">03</b></span></div>
        </div>
      </section>
      {dialog === "workspace" && <Dialog title="Choose a workspace" subtitle="Create a new account or open an existing account by its ID." onClose={() => setDialog(null)}><form className="dialogForm" onSubmit={createWorkspace}><Field label="New workspace name" name="name" placeholder="Example: Acme AI" required/><Field label="Workspace slug" name="slug" placeholder="acme-ai"/><button className="primary" disabled={loading}>{loading ? "Creating…" : "Create workspace"}</button></form><div className="dialogDivider"><span>or open existing</span></div><form className="dialogForm compactForm" onSubmit={openWorkspace}><Field label="Account ID" name="account_id" placeholder="acc_…" required/><button className="secondaryButton" disabled={loading}>Open workspace</button></form></Dialog>}
    </main>;
  }

  const { metrics, knowledge } = dashboard;
  return <main className="shell">
    <style jsx global>{`.tableRow a,.compactList a,.activityList a{color:#dce4f1;font-size:10px;font-weight:750;text-decoration:none}.tableRow a:hover,.compactList a:hover,.activityList a:hover{color:#b9abff}.projectTable .tableRow>span:last-child a{display:block;margin-top:4px;color:#a999fa;font-size:8.5px}.activityList article>div>a{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}`}</style>
    <BrandBar accountName={dashboard.account.name} onSwitch={() => { setDashboard(null); setDialog("workspace"); }} />
    <div className="appFrame">
      <aside className="sidebar">
        <nav aria-label="Control Center sections">{(["overview","projects","ai","reports","certificates","knowledge"] as Tab[]).map(item => <button key={item} className={tab === item ? "active" : ""} aria-current={tab === item ? "page" : undefined} onClick={() => selectTab(item)}><Icon name={item}/><span>{tabLabel(item)}</span>{item === "reports" && dashboard.reports.length > 0 && <em>{dashboard.reports.length}</em>}{item === "certificates" && dashboard.certificates.length > 0 && <em>{dashboard.certificates.length}</em>}</button>)}</nav>
        <div className="sidebarBottom"><Link href="/run"><Icon name="debug"/><span>Run debugger</span></Link><a href={`${API}/docs`} target="_blank" rel="noreferrer"><Icon name="docs"/><span>API reference</span></a></div>
      </aside>

      <section className="content">
        <header className="pageHeader"><div><p className="eyebrow">{tab === "overview" ? "Account overview" : tabLabel(tab)}</p><h1>{headline(tab)}</h1><p>{subheadline(tab)}</p></div><div className="headerActions"><button className="secondaryButton" onClick={() => void load()} disabled={loading}>{loading ? "Refreshing…" : "Refresh"}</button><button className="secondaryButton" onClick={() => { setConnectionSetup(null); setDialog("connect"); }} disabled={!dashboard.projects.some(project => project.connection_status !== "active")}>Connect CI</button><button className="primary" onClick={() => setDialog("project")}>Add project</button></div></header>
        {error && <p className="errorBanner">{error}</p>}

        {tab === "overview" && <ActionCenter dashboard={dashboard} onNavigate={selectTab} onConnect={() => { setConnectionSetup(null); setDialog("connect"); }} />}

        {(tab === "overview" || tab === "knowledge") && <section className="metricGrid fiveMetrics">
          <MetricCard label="Knowledge coverage" value={`${metrics.knowledge_coverage_pct}%`} detail={`${knowledge.supported_claims} of ${knowledge.claims} claims supported`} tone="violet"/>
          <MetricCard label="Connected projects" value={`${metrics.connected_projects}/${metrics.projects}`} detail={`${metrics.reports} published reports`} tone="blue"/>
          <MetricCard label="AI systems" value={String(metrics.ai_systems).padStart(2,"0")} detail={`${dashboard.ai_systems.filter(item => item.certification_status === "valid").length} currently certified`} tone="cyan"/>
          <MetricCard label="Valid certificates" value={String(metrics.valid_certificates).padStart(2,"0")} detail={`Across ${metrics.projects} registered projects`} tone="green"/>
          <MetricCard label="Needs attention" value={String(metrics.attention_items).padStart(2,"0")} detail={`${knowledge.open_unknowns} unknowns · ${knowledge.contradicted_claims} contradictions`} tone="amber"/>
        </section>}

        {tab === "overview" && <>
          <section className="overviewGrid">
            <Panel title="Project assurance" action={<button className="textButton" onClick={() => selectTab("projects")}>View all projects →</button>}><ProjectTable projects={dashboard.projects.slice(0,5)}/></Panel>
            <Panel title="Knowledge health" action={<span className="liveDot">Live account view</span>}><KnowledgeChart knowledge={knowledge} coverage={metrics.knowledge_coverage_pct}/></Panel>
          </section>
          <section className="overviewGrid lowerOverview">
            <Panel title="AI certification register" action={<button className="textButton" onClick={() => setDialog("ai")}>+ Register AI usage</button>}><AIList systems={dashboard.ai_systems.slice(0,4)}/></Panel>
            <Panel title="Latest connected reports" action={<button className="textButton" onClick={() => selectTab("reports")}>View report ledger →</button>}><ReportList reports={dashboard.reports.slice(0,5)}/></Panel>
          </section>
          <Panel title="Recent evidence activity" action={<span className="panelMeta">Latest 12 events</span>}><ActivityList activity={dashboard.activity.slice(0,6)}/></Panel>
        </>}

        {tab === "projects" && <Panel title={`${dashboard.projects.length} registered projects`} action={<button className="primary small" onClick={() => setDialog("project")}>Register project</button>}><ProjectTable projects={dashboard.projects}/></Panel>}
        {tab === "ai" && <Panel title={`${dashboard.ai_systems.length} registered AI systems`} action={<button className="primary small" onClick={() => setDialog("ai")} disabled={!dashboard.projects.length}>Register AI usage</button>}><AIRegistry systems={dashboard.ai_systems}/></Panel>}
        {tab === "reports" && <Panel title="Connected project report ledger" action={<span className="panelMeta">Authenticated CI publications</span>}><ReportTable reports={dashboard.reports} projects={dashboard.projects}/></Panel>}
        {tab === "certificates" && <Panel title="Decision certificate ledger" action={<span className="panelMeta">SHA-256 integrity proofs</span>}><CertificateTable certificates={dashboard.certificates}/></Panel>}
        {tab === "knowledge" && <section className="knowledgeLayout"><Panel title="Account knowledge composition"><KnowledgeChart knowledge={knowledge} coverage={metrics.knowledge_coverage_pct}/></Panel><Panel title="Knowledge by project"><KnowledgeProjects projects={dashboard.projects}/></Panel></section>}
      </section>
    </div>

    {dialog === "project" && <Dialog title="Register a project" subtitle="Connect a repository to account-level knowledge and certification tracking." onClose={() => setDialog(null)}><form className="dialogForm" onSubmit={createProject}><Field label="Project name" name="name" placeholder="Food Lens" required/><Field label="Repository" name="repository" placeholder="owner/repository"/><Field label="Owner" name="owner" placeholder="Trust & Safety"/><button className="primary" disabled={loading}>{loading ? "Registering…" : "Register project"}</button></form></Dialog>}
    {dialog === "ai" && <Dialog title="Register AI usage" subtitle="Certification applies to this declared usage—not to a model in every context." onClose={() => setDialog(null)}><form className="dialogForm twoColumns" onSubmit={createAISystem}><label>Project<select name="project_id" required>{dashboard.projects.map(project => <option key={project.id} value={project.id}>{project.name}</option>)}</select></label><Field label="System name" name="name" placeholder="Food image analyzer" required/><Field label="Provider" name="provider" placeholder="OpenAI" required/><Field label="Model / version" name="model" placeholder="gpt-5.6" required/><label className="fullField">Purpose<textarea name="purpose" placeholder="What decision does this AI usage support?" required/></label><Field label="Data classes" name="data_classes" placeholder="user_image, metadata"/><Field label="Enabled tools" name="tools" placeholder="web_search, internal_api"/><Field label="Owner" name="owner" placeholder="Applied AI team"/><button className="primary fullField" disabled={loading}>{loading ? "Registering…" : "Register AI usage"}</button></form></Dialog>}
    {dialog === "connect" && <Dialog title="Connect a project" subtitle="Create a project-scoped ingest token and add the generated step to GitHub Actions." onClose={() => { setDialog(null); setConnectionSetup(null); }}>{connectionSetup ? <ConnectionInstructions setup={connectionSetup}/> : <form className="dialogForm" onSubmit={createConnection}><label>Project<select name="project_id" required>{dashboard.projects.filter(project => project.connection_status !== "active").map(project => <option key={project.id} value={project.id}>{project.name}</option>)}</select></label><Field label="Repository" name="repository" placeholder="owner/repository" required/><label>Control Center endpoint<input name="endpoint" defaultValue={API} required/></label><button className="primary" disabled={loading}>{loading ? "Connecting…" : "Create connection"}</button></form>}</Dialog>}
  </main>;
}

function BrandBar({accountName,onSwitch}:{accountName:string;onSwitch:()=>void}) { return <header className="brandBar"><Link href="/" className="brand"><span className="brandGlyph">E</span><span><b>Epistemic</b><small>Control center</small></span></Link><div className="accountArea"><button className="accountSwitch" aria-label={`Switch workspace. Current workspace: ${accountName}`} onClick={onSwitch}><span className="accountAvatar">{accountName[0]?.toUpperCase() ?? "E"}</span><span><small>Workspace</small><b>{accountName}</b></span><i aria-hidden="true">⌄</i></button></div></header> }
function MetricCard({label,value,detail,tone}:{label:string;value:string;detail:string;tone:string}) { return <article className={`metricCard ${tone}`}><div><span>{label}</span><Icon name={tone}/></div><strong>{value}</strong><p>{detail}</p><i className="metricLine"/></article> }
function Panel({title,action,children}:{title:string;action?:ReactNode;children:ReactNode}) { return <section className="panel"><header><h2>{title}</h2>{action}</header>{children}</section> }
function StatusPill({status}:{status:Status}) { return <span className={`statusPill ${status}`}><i/>{status}</span> }

function ActionCenter({dashboard,onNavigate,onConnect}:{dashboard:Dashboard;onNavigate:(tab:Tab)=>void;onConnect:()=>void}) {
  const disconnected = dashboard.projects.filter(project => project.connection_status !== "active").length;
  const uncertified = dashboard.ai_systems.filter(system => system.certification_status !== "valid").length;
  const attention = dashboard.metrics.attention_items;
  const clear = disconnected === 0 && uncertified === 0 && attention === 0;
  return <section className={`actionCenter ${clear ? "clear" : "needsAction"}`} aria-labelledby="action-center-title"><div className="actionCenterIntro"><span>{clear ? "✓" : "!"}</span><div><p className="eyebrow">Recommended next step</p><h2 id="action-center-title">{clear ? "Your portfolio is ready for review" : "Focus on the items that can change a decision"}</h2><p>{clear ? "Projects are connected and current AI usage has valid certification." : "The Control Center has prioritized the shortest path to a clearer assurance state."}</p></div></div>{!clear && <div className="actionList">{attention > 0 && <button onClick={() => onNavigate("knowledge")}><b>Review evidence gaps</b><span>{attention} items need attention</span><i>→</i></button>}{uncertified > 0 && <button onClick={() => onNavigate("ai")}><b>Review uncertified AI usage</b><span>{uncertified} systems need a valid decision</span><i>→</i></button>}{disconnected > 0 && <button onClick={onConnect}><b>Connect project CI</b><span>{disconnected} projects cannot publish evidence</span><i>→</i></button>}</div>}</section>;
}

function ProjectTable({projects}:{projects:Project[]}) { if (!projects.length) return <Empty title="No projects registered" detail="Register a project to start collecting epistemic knowledge."/>; return <div className="dataTable projectTable"><div className="tableHead"><span>Project</span><span>Connection</span><span>Knowledge</span><span>Reports</span><span>Certificate</span><span>Last activity</span></div>{projects.map(project => <div className="tableRow" key={project.id}><span className="projectIdentity"><i>{initials(project.name)}</i><span><b>{project.name}</b><small>{project.repository || project.owner || "No repository connected"} · {project.ai_systems} AI systems</small></span></span><span><span className={`connectionPill ${project.connection_status}`}><i/>{project.connection_status}</span><small>{project.connection_status === "active" ? "Authenticated ingest" : "No CI publisher"}</small></span><span><b>{project.knowledge_coverage_pct}%</b><small>{project.supported_claims}/{project.claims} supported</small></span><span><b>{project.reports}</b><small>{project.runs} internal runs</small></span><span><StatusPill status={project.certification_status}/></span><span><b>{project.last_activity_at ? relativeTime(project.last_activity_at) : "—"}</b>{project.latest_run_id ? <Link href={`/run?run=${project.latest_run_id}`}>Inspect latest run →</Link> : <small>No runs</small>}</span></div>)}</div> }
function AIList({systems}:{systems:AISystem[]}) { if (!systems.length) return <Empty title="No AI usage registered" detail="Declare each model usage before asking the Engine to certify it."/>; return <div className="compactList">{systems.map(system => <article key={system.id}><span className="providerMark">{system.provider.slice(0,2).toUpperCase()}</span><div><b>{system.name}</b><small>{system.project_name} · {system.provider} {system.model}</small></div><StatusPill status={system.certification_status}/></article>)}</div> }
function ReportList({reports}:{reports:Report[]}) { if (!reports.length) return <Empty title="No connected reports" detail="Connect a project and enable publishing in its Epistemic GitHub Action."/>; return <div className="compactList reportList">{reports.map(report => <article key={report.id}><span className={`reportMark ${report.status}`}>{report.status === "passed" ? "✓" : "!"}</span><div>{report.details?.run_id ? <Link href={`/run?run=${report.details.run_id}`}>{report.tool}</Link> : <b>{report.tool}</b>}<small>{report.repository || report.workflow || "Connected project"} · {report.summary}</small></div><span className="listTime">{relativeTime(report.received_at)}</span></article>)}</div> }
function AIRegistry({systems}:{systems:AISystem[]}) { if (!systems.length) return <Empty title="Your AI register is empty" detail="Register model purpose, data, tools, and ownership to make usage certifiable."/>; return <div className="aiCards">{systems.map(system => <article className="aiCard" key={system.id}><header><span className="providerMark large">{system.provider.slice(0,2).toUpperCase()}</span><StatusPill status={system.certification_status}/></header><h3>{system.name}</h3><p>{system.purpose}</p><dl><div><dt>Project</dt><dd>{system.project_name}</dd></div><div><dt>Model</dt><dd>{system.provider} / {system.model}</dd></div><div><dt>Data</dt><dd>{system.data_classes.join(", ") || "Not declared"}</dd></div><div><dt>Tools</dt><dd>{system.tools.join(", ") || "None"}</dd></div></dl><footer><span>{system.owner || "No owner"}</span><span>{system.last_evaluated_at ? `Evaluated ${relativeTime(system.last_evaluated_at)}` : "Not evaluated"}</span></footer></article>)}</div> }
function CertificateTable({certificates}:{certificates:Certificate[]}) { if (!certificates.length) return <Empty title="No certificates issued" detail="Certificates appear after a linked project run completes policy evaluation."/>; return <div className="certificateTable"><div className="tableHead"><span>Project / AI usage</span><span>Verdict</span><span>Policy</span><span>Issued</span><span>Proof</span></div>{certificates.map(certificate => <div className="tableRow" key={`${certificate.decision_id}-${certificate.source ?? "internal"}`}><span>{certificate.source === "connected-project" ? <b>{certificate.project_name || "Unassigned project"}</b> : <Link href={`/run?run=${certificate.run_id}`}>{certificate.project_name || "Unassigned project"}</Link>}<small>{certificate.ai_system_name || certificate.run_id}</small></span><span><StatusPill status={certificate.action_allowed ? "valid" : "blocked"}/><small>{certificate.verdict}</small></span><span><b>{certificate.policy_version}</b><small>{certificate.source || "control-plane"}</small></span><span><b>{relativeTime(certificate.issued_at)}</b><small>{new Date(certificate.issued_at).toLocaleDateString()}</small></span><span><code>{certificate.digest.slice(0,18)}…</code><small>SHA-256</small></span></div>)}</div> }
function ReportTable({reports,projects}:{reports:Report[];projects:Project[]}) { if (!reports.length) return <Empty title="No reports received" detail="Connected CI workflows publish their portable report and certificate here."/>; const names=new Map(projects.map(project=>[project.id,project.name])); return <div className="reportTable"><div className="tableHead"><span>Project / tool</span><span>Status</span><span>Source</span><span>Commit</span><span>Received</span></div>{reports.map(report=><div className="tableRow" key={report.id}><span>{report.details?.run_id ? <Link href={`/run?run=${report.details.run_id}`}>{names.get(report.project_id) || report.repository || "Project"}</Link> : <b>{names.get(report.project_id) || report.repository || "Project"}</b>}<small>{report.tool}</small></span><span><span className={`reportStatus ${report.status}`}>{report.status}</span><small>exit {report.exit_code}</small></span><span><b>{report.workflow || "GitHub Actions"}</b><small>{report.branch || "—"}</small></span><span><code>{report.commit_sha ? report.commit_sha.slice(0,12) : "—"}</code><small>{report.external_id}</small></span><span><b>{relativeTime(report.received_at)}</b>{report.run_url ? <a href={report.run_url} target="_blank" rel="noreferrer">Open workflow ↗</a> : report.details?.run_id ? <Link href={`/run?run=${report.details.run_id}`}>Inspect evidence →</Link> : <small>No run URL</small>}</span></div>)}</div> }
function KnowledgeChart({knowledge,coverage}:{knowledge:Knowledge;coverage:number}) { const remaining = Math.max(knowledge.claims - knowledge.supported_claims - knowledge.contradicted_claims - knowledge.stale_claims,0); return <div className="knowledgeChart"><div className="coverageRing" style={{"--coverage":`${coverage * 3.6}deg`} as React.CSSProperties}><div><strong>{coverage}%</strong><span>supported</span></div></div><div className="legend"><Legend color="green" label="Supported" value={knowledge.supported_claims}/><Legend color="red" label="Contradicted" value={knowledge.contradicted_claims}/><Legend color="amber" label="Open / pending" value={remaining}/><Legend color="gray" label="Stale" value={knowledge.stale_claims}/><div className="evidenceTotal"><span>Evidence artifacts</span><strong>{knowledge.evidence_artifacts}</strong></div></div></div> }
function Legend({color,label,value}:{color:string;label:string;value:number}) { return <div className="legendRow"><i className={color}/><span>{label}</span><b>{value}</b></div> }
function ActivityList({activity}:{activity:Activity[]}) { if (!activity.length) return <Empty title="No activity yet" detail="Runs, AI registrations, and certificates will appear here."/>; return <div className="activityList">{activity.map(item => <article key={`${item.kind}-${item.id}`}><span className={`activityIcon ${item.kind}`}><Icon name={item.kind}/></span><div>{item.kind === "run" && item.run_id ? <Link href={`/run?run=${item.run_id}`}>{item.title}</Link> : <b>{item.title}</b>}<small>{item.detail}</small></div><span><b>{relativeTime(item.occurred_at)}</b><small>{item.status}</small></span></article>)}</div> }
function KnowledgeProjects({projects}:{projects:Project[]}) { if (!projects.length) return <Empty title="No project knowledge" detail="Evidence becomes reusable account knowledge after project runs are analyzed."/>; return <div className="knowledgeProjects">{projects.map(project => <article key={project.id}><div><b>{project.name}</b><span>{project.supported_claims} of {project.claims} claims supported</span></div><strong>{project.knowledge_coverage_pct}%</strong><i><span style={{width:`${project.knowledge_coverage_pct}%`}}/></i></article>)}</div> }
function Empty({title,detail}:{title:string;detail:string}) { return <div className="emptyPanel"><span>◎</span><b>{title}</b><p>{detail}</p></div> }

function Dialog({title,subtitle,onClose,children}:{title:string;subtitle:string;onClose:()=>void;children:ReactNode}) { return <div className="dialogBackdrop" onMouseDown={event => event.target === event.currentTarget && onClose()}><section className="dialog" role="dialog" aria-modal="true" aria-label={title}><header><div><h2>{title}</h2><p>{subtitle}</p></div><button className="dialogClose" aria-label="Close dialog" onClick={onClose}>×</button></header>{children}</section></div> }
function ConnectionInstructions({setup}:{setup:ConnectionSetup}) { return <div className="connectionInstructions"><p className="successNotice">Connection created for <b>{setup.connection.repository}</b>. The token is shown only now.</p><label>GitHub secret: EPISTEMIC_INGEST_TOKEN<div className="copyField"><code>{setup.token}</code><button onClick={() => void navigator.clipboard.writeText(setup.token)}>Copy</button></div></label><label>Workflow step<div className="codeBlock"><pre>{setup.workflow}</pre><button onClick={() => void navigator.clipboard.writeText(setup.workflow)}>Copy YAML</button></div></label><p className="securityNote">Store the token as a GitHub Actions secret. Never commit it to the repository.</p></div> }
function Field({label,name,placeholder,required=false}:{label:string;name:string;placeholder:string;required?:boolean}) { return <label>{label}<input name={name} placeholder={placeholder} required={required}/></label> }

function Icon({name}:{name:string}) { const paths:Record<string,string>={overview:"M4 13h6V4H4v9Zm0 7h6v-4H4v4Zm10 0h6v-9h-6v9Zm0-16v4h6V4h-6Z",projects:"M3 6h7l2 2h9v11H3V6Z",ai:"M8 3h8v3h3v12h-3v3H8v-3H5V6h3V3Zm1 7v4m6-4v4M9 17h6",reports:"M5 3h14v18H5V3Zm3 5h8m-8 4h8m-8 4h5",certificates:"M12 3 4 6v5c0 5 3.5 8.5 8 10 4.5-1.5 8-5 8-10V6l-8-3Zm-3 9 2 2 4-5",knowledge:"M5 4h11a3 3 0 0 1 3 3v13H7a2 2 0 0 1-2-2V4Zm0 13c0-1 1-2 2-2h12",debug:"M8 9h8m-8 4h8m-5-9h2m-8 7H2m20 0h-3m-2-5 2-2M7 6 5 4m12 12 2 2M7 16l-2 2",docs:"M5 3h10l4 4v14H5V3Zm9 0v5h5M8 12h8m-8 4h8",bell:"M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9Zm-8 12h4",violet:"M12 2 4 6v6c0 5 3 8 8 10 5-2 8-5 8-10V6l-8-4Zm-3 10 2 2 4-5",blue:"M4 12h5l2-3 2 6 2-3h5",cyan:"M8 4h8v4h4v8h-4v4H8v-4H4V8h4V4Zm2 7v3m4-3v3",green:"M4 12 9 17 20 6",amber:"M12 3 2 21h20L12 3Zm0 6v5m0 3v1",certificate:"M12 3 4 6v5c0 5 3 8 8 10 5-2 8-5 8-10V6l-8-3Z",report:"M5 3h14v18H5V3Zm3 6h8m-8 4h6",run:"M5 4h14v16H5V4Zm3 5h8m-8 4h6",ai_system:"M8 4h8v4h4v8h-4v4H8v-4H4V8h4V4Z"}; return <svg viewBox="0 0 24 24" aria-hidden="true"><path d={paths[name] ?? paths.overview}/></svg> }

function jsonRequest(body: Record<string, unknown>): RequestInit { return { method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify(body) }; }
function splitList(value: FormDataEntryValue | null) { return String(value ?? "").split(",").map(item => item.trim()).filter(Boolean); }
function message(reason: unknown) { return reason instanceof Error ? reason.message : "Request failed"; }
function initials(value:string) { return value.split(/\s+/).slice(0,2).map(word => word[0]).join("").toUpperCase(); }
function relativeTime(value:string) { const seconds=Math.round((new Date(value).getTime()-Date.now())/1000); const formatter=new Intl.RelativeTimeFormat("en",{numeric:"auto"}); const units:[Intl.RelativeTimeFormatUnit,number][]=[["year",31536000],["month",2592000],["day",86400],["hour",3600],["minute",60]]; for(const[unit,size]of units){if(Math.abs(seconds)>=size)return formatter.format(Math.round(seconds/size),unit)} return "just now"; }
function isTab(value:string|null):value is Tab { return value !== null && ["overview","projects","ai","reports","certificates","knowledge"].includes(value); }
function tabLabel(tab:Tab) { return ({overview:"Overview",projects:"Projects",ai:"AI systems",reports:"CI reports",certificates:"Decisions",knowledge:"Evidence health"})[tab]; }
function headline(tab:Tab) { return ({overview:"What needs attention now",projects:"Projects and connections",ai:"Registered AI systems",reports:"Published CI evidence",certificates:"Decision history",knowledge:"Evidence health"})[tab]; }
function subheadline(tab:Tab) { return ({overview:"See what is ready, what is blocked, and the next action to take.",projects:"Understand which repositories publish evidence and where assurance is incomplete.",ai:"Review each declared model purpose, data access, owner, and certification state.",reports:"Trace authenticated checks back to commits, workflows, and evidence summaries.",certificates:"Read human outcomes first, with immutable machine proof available for audit.",knowledge:"Find supported claims, contradictions, stale evidence, and unresolved unknowns."})[tab]; }
