. "$env:VM\lib.ps1"
python -m pip install -q comtypes
BuildCheck roles "$env:RUNNER_TEMP\roles.exe"
$p = StartApp "$env:RUNNER_TEMP\roles.exe"
Shot roles
python "$env:VM\roles.py" $env:VM 'Roles check' 'Add message' 'Show error' 'Add message' 'Add message' 'Show error'
Shot roles-after
Stop-Process $p -Force
"== Narrator smoke"
$p = StartApp "$env:RUNNER_TEMP\roles.exe"
$n = Start-Process "$env:WINDIR\System32\Narrator.exe" -PassThru; Start-Sleep 6
FrontWindow $p.MainWindowHandle; Start-Sleep 1
python "$env:VM\roles.py" $env:VM 'Roles check' 'Add message' 'Show error'
"app alive with Narrator: $(-not $p.HasExited)"
Get-Process Narrator -ErrorAction SilentlyContinue | Stop-Process -Force
Stop-Process $p -Force
"== tests"
go test ./internal/windows -run 'TestUIALandmarks|TestUIAArticle' -v 2>&1 | Select-Object -Last 15
$env:MYGO_E2E = 1; go test ./internal/e2e -run Access -v 2>&1 | Select-String -Pattern '^(=== RUN|--- |ok|FAIL|PASS)' | Select-Object -Last 60
