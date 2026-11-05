. "$env:VM\lib.ps1"
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
  Key $VK.DOWN; Key $VK.DOWN; Key $VK.DOWN; Key $VK.DOWN; Key $VK.RIGHT; Start-Sleep -Milliseconds 700; Shot "$tag-submenu"
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
  # The tray icon, right-clicked where it is.
  $line = Get-Content "$env:OUT\menus-stdout.txt" | Where-Object { $_ -like 'tray *' } | Select-Object -First 1
  "tray bounds: $line"
  if ($line) {
    $t = $line.Split(' '); $tx = [int]$t[1] + [int]$t[3] / 2; $ty = [int]$t[2] + [int]$t[4] / 2
    Shot "$tag-taskbar"
    Click $tx $ty -right
    Burst "$tag-tray"; Start-Sleep -Milliseconds 600; Shot "$tag-tray" -cursor
    Key $VK.ESC; Start-Sleep -Milliseconds 400
  }
  Stop-Process $p -Force; Start-Sleep 1
  Copy-Item "$env:OUT\menus-stdout.txt" "$env:OUT\$tag-stdout.txt" -ErrorAction SilentlyContinue
}
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
