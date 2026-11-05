. "$env:VM\lib.ps1"
python -m pip install -q comtypes
BuildCheck menus "$env:RUNNER_TEMP\menus.exe"
$pers = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize'
"theme settings before:"; Get-ItemProperty $pers -ErrorAction SilentlyContinue | Select-Object AppsUseLightTheme, SystemUsesLightTheme, EnableTransparency | Format-List | Out-String
function Burst($name) { for ($i = 0; $i -lt 5; $i++) { Shot "$name-burst$i" | Out-Null } }
function Round($tag, $appTheme) {
  "== round $tag (app $appTheme)"
  $p = StartApp "$env:RUNNER_TEMP\menus.exe" @($appTheme) 4
  Shot "$tag-window"
  # Menu bar dropdown and a submenu.
  FrontWindow $p.MainWindowHandle; Start-Sleep -Milliseconds 400
  KeyDown $VK.ALT; Key 0x46; KeyUp $VK.ALT
  Burst "$tag-menubar"; Start-Sleep -Milliseconds 600; Shot "$tag-menubar"
  Key $VK.DOWN; Key $VK.DOWN; Key $VK.DOWN; Key $VK.RIGHT; Start-Sleep -Milliseconds 700; Shot "$tag-submenu"
  Key $VK.ESC; Key $VK.ESC; Key $VK.ESC; Start-Sleep -Milliseconds 400
  # Context menu, from the button, at the pointer.
  FrontWindow $p.MainWindowHandle; Start-Sleep -Milliseconds 300
  $r = New-Object System.Drawing.Rectangle
  Add-Type -Namespace U -Name R -MemberDefinition '[DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out System.Drawing.Rectangle r);' -ErrorAction SilentlyContinue
  [void][U.R]::GetWindowRect($p.MainWindowHandle, [ref]$r)  # Left, Top, Right, Bottom in X, Y, Width, Height
  $cx = [int](($r.X + $r.Width) / 2); $cy = [int](($r.Y + $r.Height) / 2) + 30
  "window rect $($r.X),$($r.Y)-$($r.Width),$($r.Height)"
  Key $VK.TAB; Key $VK.SPACE
  Burst "$tag-context"; Start-Sleep -Milliseconds 600; Shot "$tag-context" -cursor
  Key $VK.ESC; Start-Sleep -Milliseconds 400
  # Alt+Space stays Windows' own menu.
  KeyDown $VK.ALT; Key $VK.SPACE; KeyUp $VK.ALT; Start-Sleep -Milliseconds 700; Shot "$tag-altspace"; Key $VK.ESC
  # The tray icon on the taskbar, right-clicked where it is.
  $line = Get-Content "$env:OUT\menus-stdout.txt" | Where-Object { $_ -like 'tray *' } | Select-Object -First 1
  "tray bounds: $line"
  $icon = python "$env:VM\find.py" $env:VM 'Menus check' | Where-Object { $_ -like 'rect|*|Button|*' } | Select-Object -First 1
  "tray button: $icon"
  if ($icon) {
    $f = $icon.Split('|'); $tx = [int]$f[3] + [int]$f[5] / 2; $ty = [int]$f[4] + [int]$f[6] / 2
    MouseTo $tx $ty; Start-Sleep -Milliseconds 300
    Shot "$tag-taskbar"
    Click $tx $ty -right
    Burst "$tag-tray"; Start-Sleep -Milliseconds 600; Shot "$tag-tray" -cursor
    Key $VK.ESC; Start-Sleep -Milliseconds 400
  }
  Stop-Process $p -Force; Start-Sleep 1
  Copy-Item "$env:OUT\menus-stdout.txt" "$env:OUT\$tag-stdout.txt" -ErrorAction SilentlyContinue
}
"== the icon in the overflow"
function Promote($v) {
  Get-ChildItem 'HKCU:\Control Panel\NotifyIconSettings' -ErrorAction SilentlyContinue | ForEach-Object {
    $k = Get-ItemProperty $_.PSPath
    if ($k.ExecutablePath -like '*menus.exe') { Set-ItemProperty $_.PSPath IsPromoted $v; "IsPromoted=$v $($_.PSChildName)" }
  }
}
$p = StartApp "$env:RUNNER_TEMP\menus.exe" @('light') 4
Shot overflow-first
Stop-Process $p -Force; Start-Sleep 1
Promote 0
$p = StartApp "$env:RUNNER_TEMP\menus.exe" @('light') 4
Shot overflow-hidden
"buttons near the taskbar:"
python "$env:VM\find.py" $env:VM '' | Where-Object { $_ -like 'rect|*|Button|*' } | Where-Object { [int]($_.Split('|')[4]) -ge 700 }
$chev = python "$env:VM\find.py" $env:VM 'idden' | Where-Object { $_ -like 'rect|*|Button|*' } | Select-Object -First 1
"chevron: $chev"
if ($chev) { $f = $chev.Split('|'); $cx = [int]$f[3] + [int]$f[5] / 2; $cy = [int]$f[4] + [int]$f[6] / 2 } else { $cx = 780; $cy = 744 }
Click $cx $cy; Start-Sleep 1.5
Shot overflow-open
$icon = python "$env:VM\find.py" $env:VM 'Menus check' | Where-Object { $_ -like 'rect|*|Button|*' } | Select-Object -First 1
"overflow icon: $icon"
if ($icon) {
  $f = $icon.Split('|'); $ox = [int]$f[3] + [int]$f[5] / 2 + 3; $oy = [int]$f[4] + [int]$f[6] / 2 + 2
  Click $ox $oy -right; Burst overflow-tray; Start-Sleep -Milliseconds 600; Shot overflow-tray -cursor
  "right-clicked at $ox,$oy"; Key $VK.ESC; Key $VK.ESC
}
Get-Content "$env:OUT\menus-stdout.txt" | Where-Object { $_ -like 'tray *' }
Stop-Process $p -Force; Start-Sleep 1
Promote 1
Round light-on-light light
Round dark-on-light dark
Set-ItemProperty $pers AppsUseLightTheme 0; Set-ItemProperty $pers SystemUsesLightTheme 0; Start-Sleep 3
Round light-on-dark light
Round system-dark system
Set-ItemProperty $pers EnableTransparency 0; Start-Sleep 3
Round notransparency-dark system
Set-ItemProperty $pers AppsUseLightTheme 1; Set-ItemProperty $pers SystemUsesLightTheme 1
Round notransparency-light light
Set-ItemProperty $pers EnableTransparency 1
"== tests"
go test ./internal/windows/... 2>&1 | Select-Object -Last 15
$env:MYGO_E2E = 1; go test ./internal/e2e -run 'TestNativeMenuShortcuts|TestPageMenuPaste|TestMenuIconsWindows' -v 2>&1 | Select-String -Pattern '^(=== RUN|--- |ok|FAIL|PASS|\s+\S+_test.go)' | Select-Object -Last 40
