$ErrorActionPreference = "Stop"

$pairs = @(
    @{ Example = "services/api/.env.example"; Target = "services/api/.env" },
    @{ Example = "apps/extension/.env.example"; Target = "apps/extension/.env" },
    @{ Example = "apps/dashboard/.env.example"; Target = "apps/dashboard/.env" }
)

$repoRoot = Join-Path $PSScriptRoot "../.."

foreach ($pair in $pairs) {
    $example = Join-Path $repoRoot $pair.Example
    $target = Join-Path $repoRoot $pair.Target

    if (Test-Path $target) {
        Write-Host "exists $($pair.Target)"
        continue
    }

    Copy-Item $example $target
    Write-Host "created $($pair.Target)"
}
