. "$env:VM\lib.ps1"
BuildCheck held "$env:RUNNER_TEMP\held.exe"
$log = "$env:OUT\held-stdout.txt"
function Mark($s) { "---- $s"; Add-Content "$env:OUT\marks.txt" "$((Get-Date).ToString('HH:mm:ss.fff')) $s" }
function Last { $l = Get-Content $log | Select-Object -Last 1; "   app: $l" }
function Front { [void][W]::SetForegroundWindow($p.MainWindowHandle); Start-Sleep -Milliseconds 500 }
$F = 0x46
$p = StartApp "$env:RUNNER_TEMP\held.exe"
Front
Mark 'typing abc'
Key 0x41; Key 0x42; Key 0x43; Start-Sleep -Milliseconds 400; Last
Shot held-typing
Mark '1a: hold F, Alt+F opens File, release F in the menu, Escape'
KeyDown $F; Start-Sleep -Milliseconds 200; Last
KeyDown $VK.ALT; KeyDown $F; KeyUp $VK.ALT; Start-Sleep -Milliseconds 600
Shot held-altf-menu; Last
KeyUp $F; Start-Sleep -Milliseconds 300; Key $VK.ESC; Key $VK.ESC; Start-Sleep -Milliseconds 500; Last
Shot held-altf-after
Mark '1b: hold Space, Alt+Space opens the system menu, release Space, Escape'
Front
KeyDown $VK.SPACE; Start-Sleep -Milliseconds 200; Last
KeyDown $VK.ALT; KeyDown $VK.SPACE; KeyUp $VK.ALT; Start-Sleep -Milliseconds 600
Shot held-altspace-menu; Last
KeyUp $VK.SPACE; Start-Sleep -Milliseconds 300; Key $VK.ESC; Key $VK.ESC; Start-Sleep -Milliseconds 500; Last
Mark '1c: hold F, F10 into the menu bar, release F, Escape'
Front
KeyDown $F; Start-Sleep -Milliseconds 200; Last
Key $VK.F10; Start-Sleep -Milliseconds 600; Key $VK.DOWN; Start-Sleep -Milliseconds 400
Shot held-f10-menu; Last
KeyUp $F; Start-Sleep -Milliseconds 300; Key $VK.ESC; Key $VK.ESC; Start-Sleep -Milliseconds 500; Last
Mark '1d: hold F, right-click the field (context menu), release F, Escape'
Front
Add-Type -Namespace U -Name R -MemberDefinition '[DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out System.Drawing.Rectangle r);' -ErrorAction SilentlyContinue
$r = New-Object System.Drawing.Rectangle; [void][U.R]::GetWindowRect($p.MainWindowHandle, [ref]$r)
"window $($r.X),$($r.Y) to $($r.Width),$($r.Height)"
KeyDown $F; Start-Sleep -Milliseconds 200; Last
Click ($r.X + 120) ($r.Y + 70) -right; Start-Sleep -Milliseconds 600
Shot held-context-menu; Last
KeyUp $F; Start-Sleep -Milliseconds 300; Key $VK.ESC; Start-Sleep -Milliseconds 500; Last
Mark '3: hold F, Alt+Tab away, release F elsewhere, come back'
$np = Start-Process notepad -PassThru; Start-Sleep 3; Front
KeyDown $F; Start-Sleep -Milliseconds 300; Last
KeyDown $VK.ALT; Key $VK.TAB; KeyUp $VK.ALT; Start-Sleep -Milliseconds 800
"foreground is app: $([W]::GetForegroundWindow() -eq $p.MainWindowHandle)"
Shot held-alttab-away; Last
KeyUp $F; Start-Sleep -Milliseconds 300; Front; Start-Sleep -Milliseconds 500; Last
Shot held-alttab-back
Get-Process notepad -ErrorAction SilentlyContinue | Stop-Process -Force
Mark '2: Japanese IME'
$before = Get-WinUserLanguageList
"languages before: $(($before | ForEach-Object { $_.LanguageTag + ' ' + ($_.InputMethodTips -join ',') }) -join '; ')"
$l = Get-WinUserLanguageList; $l.Add('ja-JP'); Set-WinUserLanguageList $l -Force
"languages now: $((Get-WinUserLanguageList | ForEach-Object { $_.LanguageTag + ' ' + ($_.InputMethodTips -join ',') }) -join '; ')"
Stop-Process $p -Force; Start-Sleep 1
$p = StartApp "$env:RUNNER_TEMP\held.exe"; Front
Add-Type -Namespace U -Name K -MemberDefinition '[DllImport("user32.dll")] public static extern IntPtr LoadKeyboardLayout(string id, uint flags); [DllImport("user32.dll")] public static extern IntPtr GetKeyboardLayout(uint tid); [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, IntPtr p); [DllImport("user32.dll")] public static extern IntPtr PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);' -ErrorAction SilentlyContinue
$hkl = [U.K]::LoadKeyboardLayout('00000411', 1)
[void][U.K]::PostMessage($p.MainWindowHandle, 0x50, [IntPtr]::Zero, $hkl)  # WM_INPUTLANGCHANGEREQUEST
Start-Sleep 1
$tid = [U.K]::GetWindowThreadProcessId($p.MainWindowHandle, [IntPtr]::Zero)
"app keyboard layout: {0:x}" -f [int64][U.K]::GetKeyboardLayout($tid)
Key 0x16  # VK_IME_ON
Start-Sleep -Milliseconds 500
Mark '2a: K down before composing, A, then K up after'
KeyDown 0x4B; Start-Sleep -Milliseconds 300; Last
Key 0x41; Start-Sleep -Milliseconds 300
KeyUp 0x4B; Start-Sleep -Milliseconds 300
Key 0x41; Start-Sleep -Milliseconds 500; Shot held-ime-composing; Last
Key $VK.ENTER; Start-Sleep -Milliseconds 500; Last
Shot held-ime-committed
Key 0x1A  # VK_IME_OFF
Mark '2b: hold F with the IME off, IME on, release F'
KeyDown $F; Start-Sleep -Milliseconds 200; Key 0x16; Start-Sleep -Milliseconds 200; Key 0x4E; Start-Sleep -Milliseconds 200; KeyUp $F; Start-Sleep -Milliseconds 400; Last
Key $VK.ESC; Key 0x1A; Start-Sleep -Milliseconds 400; Last
Set-WinUserLanguageList $before -Force
"languages restored: $((Get-WinUserLanguageList | ForEach-Object { $_.LanguageTag }) -join '; ')"
Stop-Process $p -Force
"== app log"; Get-Content $log
"== tests"
go test ./ui -run Held -v 2>&1 | Select-String -Pattern '^(=== RUN|--- |ok|FAIL|PASS)'
