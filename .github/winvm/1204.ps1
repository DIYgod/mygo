. "$env:VM\lib.ps1"
. "$env:VM\pointer.ps1"
python -m pip install -q comtypes
$clip = Get-Clipboard -Raw -ErrorAction SilentlyContinue
"clipboard before: '$clip'"
BuildCheck cursors "$env:RUNNER_TEMP\cursors.exe"
$p = StartApp "$env:RUNNER_TEMP\cursors.exe"
Shot cursors-window
$r = Rects 'Cursors check'
"elements: $($r.Keys -join ', ')"
foreach ($k in 'progress', 'wait', 'help', 'cell', 'zoom-in', 'zoom-out', 'all-scroll', 'no-drop', 'pointer', 'crosshair', 'not-allowed', 'move') {
  $e = $r["cursor $k"]
  if (-not $e) { "no element for $k"; continue }
  MouseTo $e.cx $e.cy; Start-Sleep -Milliseconds 150
  MouseTo ($e.cx + 1) $e.cy; Start-Sleep -Milliseconds 400
  CursorShot "cursor-$k"
}
"== mouse on the pad"
$pad = $r['Pad']
[void][W]::SetCursorPos($pad.x + 20, $pad.y + 20); Start-Sleep -Milliseconds 200
[W]::mouse_event(2, 0, 0, 0, [UIntPtr]::Zero); Start-Sleep -Milliseconds 100
[void][W]::SetCursorPos($pad.x + 60, $pad.y + 30); Start-Sleep -Milliseconds 100
[W]::mouse_event(4, 0, 0, 0, [UIntPtr]::Zero); Start-Sleep -Milliseconds 300
"== pen pressing harder and softer where it stands"
$px = $pad.x + 200; $py = $pad.y + 60
"pen down: " + [Ptr]::Pen($px, $py, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::DOWN, 200); Start-Sleep -Milliseconds 300
foreach ($pr in 400, 700, 1000, 500) { "pen pressure $($pr): " + [Ptr]::Pen($px, $py, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::UPDATE, $pr); Start-Sleep -Milliseconds 200 }
"pen up: " + [Ptr]::Pen($px, $py, [Ptr]::INRANGE -bor [Ptr]::UP, 0); Start-Sleep -Milliseconds 100
[void][Ptr]::Pen($px, $py, [Ptr]::UPDATE, 0); Start-Sleep -Milliseconds 300
"== pen stroke on the pad"
PenStroke ($pad.x + 30) ($pad.y + 40)
Start-Sleep 1
"== touch drag on the pad"
TouchDrag ($pad.x + 30) ($pad.y + 80)
Start-Sleep 1
"== Copy by mouse"
$copy = $r['Copy']
Click $copy.cx $copy.cy; Start-Sleep 1
"clipboard: '$(Get-Clipboard -Raw)'"
Set-Clipboard -Value 'other'
"== Copy by pen tap"
PenTap $copy.cx $copy.cy; Start-Sleep 1
"clipboard: '$(Get-Clipboard -Raw)'"
Set-Clipboard -Value 'other'
"== Copy by touch tap"
TouchTap $copy.cx $copy.cy; Start-Sleep 1
"clipboard: '$(Get-Clipboard -Raw)'"
Shot cursors-after
Stop-Process $p -Force
"== app output"; Get-Content "$env:OUT\cursors-stdout.txt"
if ($clip) { Set-Clipboard -Value $clip } else { Set-Clipboard -Value $null }
"== main, for comparison: the same pen stroke and touch drag"
BuildOnMain pointerbase "$env:RUNNER_TEMP\pointerbase.exe"
$p = StartApp "$env:RUNNER_TEMP\pointerbase.exe"
$r = Rects 'Pointer check'; $pad = $r['Pad']
PenStroke ($pad.x + 30) ($pad.y + 40) | Out-Null; Start-Sleep 1
"-- touch drag on main"
TouchDrag ($pad.x + 30) ($pad.y + 120) | Out-Null; Start-Sleep 1
Stop-Process $p -Force
Get-Content "$env:OUT\pointerbase-stdout.txt"
"== tests"
go test ./internal/windows -run 'TestPointerInfoLayout|TestDrawnCursors' -v 2>&1 | Select-String -Pattern '^(=== RUN|--- |ok|FAIL|PASS|\s+\S+_test.go)'
