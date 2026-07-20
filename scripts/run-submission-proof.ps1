$ErrorActionPreference = "Stop"

if (-not $env:OPENAI_API_KEY) {
    throw "OPENAI_API_KEY must be set in the current shell. Do not store it in the repository."
}

$root = Split-Path -Parent $PSScriptRoot
$output = Join-Path $root ".cache\submission"
$server = Join-Path $output "epistemic-control-plane-openai.exe"
$stdout = Join-Path $output "control-plane.stdout.log"
$stderr = Join-Path $output "control-plane.stderr.log"
New-Item -ItemType Directory -Force -Path $output | Out-Null

Push-Location (Join-Path $root "apps\control-plane")
try {
    go build -o $server ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "control-plane build failed" }
} finally {
    Pop-Location
}

$previousMode = $env:ANALYZER_MODE
$previousModel = $env:OPENAI_MODEL
$previousAddress = $env:CONTROL_PLANE_ADDR
$previousEndpoint = $env:EPISTEMIC_ENDPOINT
$process = $null
try {
    $env:ANALYZER_MODE = "openai"
    $env:OPENAI_MODEL = if ($previousModel) { $previousModel } else { "gpt-5.6" }
    $env:CONTROL_PLANE_ADDR = ":8081"
    $env:EPISTEMIC_ENDPOINT = "http://127.0.0.1:8081"
    $process = Start-Process -FilePath $server -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    for ($attempt = 0; $attempt -lt 60; $attempt++) {
        try {
            $health = Invoke-RestMethod -Uri "$env:EPISTEMIC_ENDPOINT/healthz" -TimeoutSec 2
            if ($health.status -eq "ok") { break }
        } catch {
            Start-Sleep -Milliseconds 500
        }
    }
    if (-not $health -or $health.status -ne "ok") { throw "OpenAI control plane did not become healthy; inspect $stderr" }

    Push-Location $root
    try {
        node scripts/submission-openai-proof.mjs
        if ($LASTEXITCODE -ne 0) { throw "GPT-5.6 proof failed" }
    } finally {
        Pop-Location
    }

    Push-Location (Join-Path $root "workers\codex-worker")
    try {
        npm ci
        if ($LASTEXITCODE -ne 0) { throw "Codex worker install failed" }
        npm run build
        if ($LASTEXITCODE -ne 0) { throw "Codex worker build failed" }
        node dist/main.js --approved --repository ../../demo/unsafe-orders-pr --specification ../../demo/verification-spec.json --output ../../.cache/submission/codex-proof.json
        if ($LASTEXITCODE -ne 0) { throw "Codex proof failed" }
    } finally {
        Pop-Location
    }
} finally {
    if ($process -and -not $process.HasExited) { Stop-Process -Id $process.Id }
    $env:ANALYZER_MODE = $previousMode
    $env:OPENAI_MODEL = $previousModel
    $env:CONTROL_PLANE_ADDR = $previousAddress
    $env:EPISTEMIC_ENDPOINT = $previousEndpoint
}

Write-Output "Submission proof complete: $output"
