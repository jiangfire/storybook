param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$Targets
)

$ErrorActionPreference = 'Stop'

if (-not $Targets -or $Targets.Count -eq 0) {
  $Targets = @('./...')
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$env:GOCACHE = Join-Path $repoRoot '.gocache_local'

$excludeRules = @(
  'internal/fileutil/workspace\.go:G304'
  'cmd/mcp/run_config\.go:G304,G117'
  'cmd/embedui/main\.go:G304'
  'internal/handler/auth_handler\.go:G117'
  'internal/handler/user_management_handler\.go:G117'
  'internal/bootstrap/admin\.go:G117'
  'internal/config/config\.go:G117'
  'internal/handler/ai_handler\.go:G117'
  'internal/service/ai_service\.go:G117'
) -join ';'

Write-Host "Running gosec with repo exclusions..." -ForegroundColor Cyan
Write-Host "--exclude-rules=$excludeRules" -ForegroundColor DarkGray

gosec --exclude-rules=$excludeRules @Targets
