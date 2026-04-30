$ErrorActionPreference = "Stop"

$pairs = @(
    @{ Example = "services/api/.env.example"; Target = "services/api/.env" },
    @{ Example = "apps/extension/.env.example"; Target = "apps/extension/.env" },
    @{ Example = "apps/dashboard/.env.example"; Target = "apps/dashboard/.env" }
)

foreach ($pair in $pairs) {
    $example = Join-Path $PSScriptRoot ".." $pair.Example
    $target = Join-Path $PSScriptRoot ".." $pair.Target

    if (Test-Path $target) {
        Write-Host "exists $($pair.Target)"
        continue
    }

    Copy-Item $example $target
    Write-Host "created $($pair.Target)"
}
