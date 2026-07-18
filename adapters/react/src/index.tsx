import React from "react";

export type PortableDecision={decision_id:string;status:"allow"|"block"|"indeterminate"|"error";action_allowed:boolean;reasons:Array<{code:string;message:string}>;conditions:string[]};
export type PortableCertificate={id:string;decision_id:string;issued_at:string;artifact_hashes:string[];proof:{algorithm:string;digest:string}};

export function DecisionBadge({decision}:{decision:PortableDecision}){const color={allow:"#1f9d68",block:"#d64545",indeterminate:"#bd8b18",error:"#d64545"}[decision.status];return <span role="status" style={{display:"inline-flex",gap:8,alignItems:"center",padding:"6px 10px",border:`1px solid ${color}`,borderRadius:6,color,fontFamily:"system-ui",fontWeight:700}}><i style={{width:8,height:8,borderRadius:"50%",background:color}}/>{decision.status.toUpperCase()}</span>}
export function CertificateView({certificate}:{certificate:PortableCertificate}){return <article style={{border:"1px solid #7765a8",borderRadius:8,padding:20,fontFamily:"system-ui"}}><small>Epistemic Decision Certificate</small><h3>{certificate.decision_id}</h3><p>Issued {new Date(certificate.issued_at).toLocaleString()}</p><code style={{wordBreak:"break-all"}}>{certificate.proof.algorithm}: {certificate.proof.digest}</code><p>{certificate.artifact_hashes.length} content-addressed artifacts</p></article>}
