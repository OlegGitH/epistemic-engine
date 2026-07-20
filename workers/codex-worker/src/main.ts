import { Codex } from "@openai/codex-sdk";
import { createHash } from "node:crypto";
import { execFile } from "node:child_process";
import { cp, mkdtemp, readFile, readdir, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, join, resolve, sep } from "node:path";
import { promisify } from "node:util";

const run = promisify(execFile);

type VerificationSpec = {
  claim_id: string;
  claim: string;
  check: string;
  required_test?: string;
};

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (!args.approved) throw new Error("--approved is required; Codex generation must follow explicit human approval");
  if (!args.repository || !args.specification || !args.output) throw new Error("--repository, --specification, and --output are required");

  const repository = resolve(args.repository);
  const specificationPath = resolve(args.specification);
  const outputPath = resolve(args.output);
  const specification = JSON.parse(await readFile(specificationPath, "utf8")) as VerificationSpec;
  const workspace = await mkdtemp(join(tmpdir(), "epistemic-codex-"));
  const copy = join(workspace, basename(repository));

  try {
    await cp(repository, copy, { recursive: true });
    await git(copy, ["init"]);
    await git(copy, ["config", "user.email", "epistemic-worker@localhost"]);
    await git(copy, ["config", "user.name", "Epistemic Worker"]);
    await git(copy, ["add", "--all"]);
    await git(copy, ["commit", "-m", "verification baseline"]);

    const prompt = [
      "Create exactly one bounded verification test for the supplied epistemic claim.",
      "Only edit files under tests/. Do not modify product code, configuration, dependencies, or CI.",
      "Do not use the network. Do not broaden the goal. Run only the targeted test if useful.",
      `Claim ID: ${specification.claim_id}`,
      `Claim: ${specification.claim}`,
      `Check: ${specification.check}`,
      specification.required_test ? `Required test file: ${specification.required_test}. Create or update exactly this test file and no other file.` : "",
    ].filter(Boolean).join("\n");

    const codex = new Codex();
    const thread = codex.startThread({
      workingDirectory: copy,
      sandboxMode: "workspace-write",
      networkAccessEnabled: false,
      webSearchMode: "disabled",
      approvalPolicy: "never",
      ...(process.env.CODEX_MODEL ? { model: process.env.CODEX_MODEL } : {}),
    });
    const result = await thread.run(prompt);
    await removeGeneratedTestArtifacts(join(copy, "tests"));
    // Intent-to-add makes new files visible to diff without staging their contents.
    // The checkout is disposable, and this lets the same scope checks cover both
    // newly created tests and modifications to existing tests.
    await git(copy, ["add", "--intent-to-add", "--all"]);
    const changed = (await git(copy, ["diff", "--name-only"])).stdout.split(/\r?\n/).filter(Boolean);
    if (changed.length === 0) {
      const summary = result.finalResponse.trim().replace(/\s+/g, " ").slice(0, 500);
      throw new Error(`Codex did not create a verification patch. Final response: ${summary || "(empty)"}`);
    }
    if (changed.some((file) => file !== "tests" && !file.startsWith(`tests${sep}`) && !file.startsWith("tests/"))) {
      throw new Error(`Codex changed files outside tests/: ${changed.join(", ")}`);
    }
    if (specification.required_test) {
      const required = specification.required_test.replaceAll("\\", "/");
      const unexpected = changed.filter((file) => file.replaceAll("\\", "/") !== required);
      if (unexpected.length > 0 || !changed.some((file) => file.replaceAll("\\", "/") === required)) {
        throw new Error(`Codex must change only the required test file ${required}; changed: ${changed.join(", ")}`);
      }
    }
    const patch = (await git(copy, ["diff", "--binary", "--", "tests"])).stdout;
    const patchHash = createHash("sha256").update(patch).digest("hex");
    const specHash = createHash("sha256").update(JSON.stringify(specification)).digest("hex");
    await writeFile(outputPath, JSON.stringify({
      worker: "openai-codex-sdk", thread_id: thread.id, specification_hash: specHash,
      changed_files: changed, patch, patch_sha256: patchHash, final_response: result.finalResponse,
      applied: false, approval_recorded: true,
    }, null, 2));
    process.stdout.write(JSON.stringify({ output: outputPath, patch_sha256: patchHash, changed_files: changed }) + "\n");
  } finally {
    await rm(workspace, { recursive: true, force: true });
  }
}

async function git(cwd: string, args: string[]) {
  return run("git", args, { cwd, maxBuffer: 2 * 1024 * 1024 });
}

async function removeGeneratedTestArtifacts(root: string): Promise<void> {
  let entries;
  try {
    entries = await readdir(root, { withFileTypes: true });
  } catch {
    return;
  }
  for (const entry of entries) {
    const path = join(root, entry.name);
    if (entry.isDirectory() && (entry.name === "__pycache__" || entry.name === ".pytest_cache")) {
      await rm(path, { recursive: true, force: true });
    } else if (entry.isDirectory()) {
      await removeGeneratedTestArtifacts(path);
    } else if (/\.(?:pyc|pyo)$/.test(entry.name)) {
      await rm(path, { force: true });
    }
  }
}

function parseArgs(values: string[]) {
  const result: { repository?: string; specification?: string; output?: string; approved: boolean } = { approved: false };
  for (let index = 0; index < values.length; index++) {
    const value = values[index];
    if (value === "--approved") result.approved = true;
    else if (value === "--repository") result.repository = values[++index];
    else if (value === "--specification") result.specification = values[++index];
    else if (value === "--output") result.output = values[++index];
  }
  for (const path of [result.repository, result.specification, result.output]) {
    if (path && path.includes("\0")) throw new Error("invalid path");
  }
  return result;
}

main().catch((error: unknown) => {
  process.stderr.write((error instanceof Error ? error.message : String(error)) + "\n");
  process.exitCode = 1;
});
