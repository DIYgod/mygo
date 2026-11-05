. "$env:VM\lib.ps1"
python -m pip install -q comtypes
BuildCheck states "$env:RUNNER_TEMP\states.exe"
$p = StartApp "$env:RUNNER_TEMP\states.exe"
Shot states
python "$env:VM\roles.py" $env:VM 'States check' 'Details' 'Day' 'Next day' 'Add message' 'Show error' 'Add message' 'Add message' 'Show error'
Shot states-after
Stop-Process $p -Force
"== Narrator smoke"
$p = StartApp "$env:RUNNER_TEMP\states.exe"
$n = Start-Process "$env:WINDIR\System32\Narrator.exe" -PassThru; Start-Sleep 6
[void][W]::SetForegroundWindow($p.MainWindowHandle); Start-Sleep 1
python "$env:VM\roles.py" $env:VM 'States check' 'Add message' 'Show error' 'Next day'
"app alive with Narrator: $(-not $p.HasExited)"
Get-Process Narrator -ErrorAction SilentlyContinue | Stop-Process -Force
Stop-Process $p -Force
"== tests"
go test ./internal/windows -run 'TestUIATreeAnnouncer|TestUIAStates' -v 2>&1 | Select-Object -Last 15
$env:MYGO_E2E = 1; go test ./internal/e2e -run 'AccessRoles|AccessStates|Access' -v 2>&1 | Select-String -Pattern '^(=== RUN|--- |ok|FAIL|PASS)' | Select-Object -Last 60
"== full tests (no CI ran on this PR)"
$env:MYGO_E2E = ''; go test ./... 2>&1 | Select-String -Pattern '^(ok|FAIL|---|panic)' | Select-Object -Last 80
