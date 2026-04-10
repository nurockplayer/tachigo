$ErrorActionPreference = "Stop"

$pairs = @(
    @{ Example = "backend/.env.example"; Target = "backend/.env" },
    @{ Example = "tachimint/.env.example"; Target = "tachimint/.env" },
    @{ Example = "dashboard/.env.example"; Target = "dashboard/.env" }
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
