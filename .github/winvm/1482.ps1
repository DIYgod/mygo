. "$env:VM\lib.ps1"
$env:PYTHONIOENCODING = 'utf-8'
python -m pip install -q comtypes
"== display language"
"override: [$(Get-WinUILanguageOverride)]"
"CurrentUICulture: $([Globalization.CultureInfo]::CurrentUICulture.Name)"
"Culture: $((Get-Culture).Name), system locale: $((Get-WinSystemLocale).Name)"
"user languages before:"; Get-WinUserLanguageList | Format-Table LanguageTag, Autonym | Out-String
"== en-US, then zh-Hans-CN"
Set-WinUserLanguageList -LanguageList (New-WinUserLanguageList en-US) -Force
$l = Get-WinUserLanguageList; $l.Add('zh-Hans-CN'); Set-WinUserLanguageList -LanguageList $l -Force
"User Profile Languages: $((Get-ItemProperty 'HKCU:\Control Panel\International\User Profile').Languages -join ', ')"
Get-WinUserLanguageList | Format-Table LanguageTag, Autonym | Out-String
"CurrentUICulture now: $([Globalization.CultureInfo]::CurrentUICulture.Name)"
function Probe($tag) {
  Copy-Item "$env:VM\hanprobe_windows_test.go" internal\text\zz_hanprobe_windows_test.go
  "== $tag tests ($(git rev-parse --short HEAD))"
  go test ./internal/text/ -run 'TestDWriteUserHan|TestZZHanProbe' -v -count=1 2>&1
  Remove-Item internal\text\zz_hanprobe_windows_test.go
}
function Gallery($tag) {
  go build -o "$env:RUNNER_TEMP\gallery-$tag.exe" ./examples/gallery
  $p = StartApp "$env:RUNNER_TEMP\gallery-$tag.exe" @() 6
  FrontWindow $p.MainWindowHandle; Start-Sleep 1
  $b = python "$env:VM\find.py" $env:VM 'Languages' | Select-Object -First 3
  "languages elements: $b"
  $b = $b | Select-Object -First 1
  if ($b) { $f = $b.Split('|'); Click ([int]$f[3] + [int]$f[5] / 2) ([int]$f[4] + [int]$f[6] / 2); Start-Sleep 2 }
  Shot "gallery-$tag-languages"
  Stop-Process $p -Force -ErrorAction SilentlyContinue; Start-Sleep 1
}
Probe pr
Gallery pr
git fetch -q --depth 1 origin main
git worktree add -q "$env:RUNNER_TEMP\main" FETCH_HEAD
Push-Location "$env:RUNNER_TEMP\main"
try { Probe main; Gallery main } finally { Pop-Location }
"== restore"
Set-WinUserLanguageList -LanguageList (New-WinUserLanguageList en-US) -Force
