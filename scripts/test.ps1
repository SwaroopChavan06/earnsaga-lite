# Run unit tests with readable per-case output.
# Format:  -: TestName   then   -> PASS | -> FAIL  << FAILED
# Skips packages that have no *_test.go.
$ErrorActionPreference = "Continue"

$packages = @(go list -f "{{if .TestGoFiles}}{{.ImportPath}}{{end}}" ./... |
    Where-Object { $_.Trim() -ne "" })

if ($packages.Count -eq 0) {
    Write-Host "No packages with tests found."
    exit 1
}

Write-Host "Testing $($packages.Count) package(s)"
Write-Host ""

$failed = New-Object System.Collections.Generic.List[string]
$passCount = 0
$failCount = 0
$currentPkg = ""

$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = "go"
$psi.Arguments = "test -v -count=1 " + ($packages -join " ")
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.UseShellExecute = $false
$psi.CreateNoWindow = $true
$psi.WorkingDirectory = (Get-Location).Path

$proc = [System.Diagnostics.Process]::Start($psi)

function Write-TestLine {
    param([string]$Line)

    if ($Line -match '^=== RUN\s+(\S+)') {
        Write-Host "-: $($Matches[1])"
        return
    }

    if ($Line -match '^\s*--- (PASS|FAIL|SKIP):\s+(\S+)\s+\(([^)]+)\)') {
        $status = $Matches[1]
        $name = $Matches[2]
        $dur = $Matches[3]

        if ($status -eq "PASS") {
            Write-Host "-> PASS  $name  ($dur)" -ForegroundColor Green
            $script:passCount++
        }
        elseif ($status -eq "SKIP") {
            Write-Host "-> SKIP  $name  ($dur)" -ForegroundColor Yellow
        }
        else {
            Write-Host "-> FAIL  $name  ($dur)  << FAILED" -ForegroundColor Red
            $script:failCount++
            $script:failed.Add("$script:currentPkg :: $name")
        }
        return
    }

    if ($Line -match '^ok\s+(\S+)') {
        $script:currentPkg = $Matches[1]
        Write-Host ""
        Write-Host "ok  $($Matches[1])" -ForegroundColor Green
        Write-Host ""
        return
    }

    if ($Line -match '^FAIL\s+(\S+)') {
        $script:currentPkg = $Matches[1]
        Write-Host ""
        Write-Host "FAIL  $($Matches[1])  << PACKAGE FAILED" -ForegroundColor Red
        Write-Host ""
        return
    }

    # Keep failure details / t.Log / panic output visible
    if ($Line -match '^\s*(Error Trace|Error:|Messages:|expected|got |panic:|--- FAIL)' -or
        $Line -match '^\s+\S+.*_test\.go:') {
        Write-Host $Line -ForegroundColor Red
        return
    }

    if ($Line -match '^(PASS|FAIL)$') {
        return
    }

    if ($Line.Trim() -ne "" -and $Line -notmatch '^===') {
        # package banner from go list path appearing mid-run
        if ($Line -match '^# ') {
            Write-Host $Line -ForegroundColor DarkGray
        }
    }
}

while (-not $proc.StandardOutput.EndOfStream) {
    $line = $proc.StandardOutput.ReadLine()
    if ($null -ne $line) { Write-TestLine $line }
}
while (-not $proc.StandardError.EndOfStream) {
    $errLine = $proc.StandardError.ReadLine()
    if ($null -ne $errLine -and $errLine.Trim() -ne "") {
        Write-Host $errLine -ForegroundColor Red
    }
}

$proc.WaitForExit()
$code = $proc.ExitCode

Write-Host "----------------------------------------"
Write-Host "Summary: $passCount passed, $failCount failed"
if ($failed.Count -gt 0) {
    Write-Host ""
    Write-Host "FAILED TESTS:" -ForegroundColor Red
    foreach ($f in $failed) {
        Write-Host "  << $f" -ForegroundColor Red
    }
}
Write-Host "----------------------------------------"

exit $code
