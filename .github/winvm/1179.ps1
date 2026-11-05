. "$env:VM\lib.ps1"
python -m pip install -q comtypes
BuildCheck listbox "$env:RUNNER_TEMP\listbox.exe"
$p = StartApp "$env:RUNNER_TEMP\listbox.exe"
Shot listbox-closed
python "$env:VM\listbox.py" $env:VM $env:OUT
Shot listbox-after
Stop-Process $p -Force
"== Narrator"
# Narrator runs once, against an open popup; its speech can't be heard here.
$p = StartApp "$env:RUNNER_TEMP\listbox.exe"
$n = Start-Process "$env:WINDIR\System32\Narrator.exe" -PassThru; Start-Sleep 6
FrontWindow $p.MainWindowHandle; Start-Sleep 1
Key $VK.TAB; Key $VK.SPACE; Start-Sleep 1; Key $VK.DOWN; Start-Sleep 2
Shot listbox-narrator
"app alive with Narrator: $(-not $p.HasExited)"
Stop-Process $n -Force -ErrorAction SilentlyContinue; Get-Process Narrator -ErrorAction SilentlyContinue | Stop-Process -Force
Stop-Process $p -Force
