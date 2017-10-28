# Whalebrew Installer for Windows

try {
  $orgErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Stop"

  $VERSION = "0.1.0"
  $BINARY_URL = "https://github.com/3846masa/whalebrew/releases/download/$VERSION/whalebrew-Windows-x86_64.exe"
  $USER_PATH = [Environment]::GetEnvironmentVariable("Path", [System.EnvironmentVariableTarget]::User)
  $WHALEBREW_INSTALL_PATH = [Environment]::GetEnvironmentVariable("WHALEBREW_INSTALL_PATH", [System.EnvironmentVariableTarget]::User)

  if ($WHALEBREW_INSTALL_PATH -eq $null -or $WHALEBREW_INSTALL_PATH -eq '') {
    $WHALEBREW_INSTALL_PATH = 'C:\whalebrew';
    [Environment]::SetEnvironmentVariable("WHALEBREW_INSTALL_PATH", $WHALEBREW_INSTALL_PATH, [System.EnvironmentVariableTarget]::User)
  }

  if (![System.IO.Directory]::Exists($WHALEBREW_INSTALL_PATH)) {
    [System.IO.Directory]::CreateDirectory($WHALEBREW_INSTALL_PATH)
  }

  if ($($USER_PATH).ToLower().Contains($($WHALEBREW_INSTALL_PATH).ToLower()) -eq $false) {
    [Environment]::SetEnvironmentVariable("Path", "$USER_PATH;%WHALEBREW_INSTALL_PATH%", [System.EnvironmentVariableTarget]::User)
  }

  Write-Output "`nDownloading Whalebrew from : $BINARY_URL"

  $WHALEBREW_PATH = Join-Path $WHALEBREW_INSTALL_PATH "whalebrew.exe"
  (New-Object System.Net.WebClient).DownloadFile($BINARY_URL, "$WHALEBREW_PATH")

  Write-Output "`nInstalled whalebrew to `"$WHALEBREW_PATH`"`n"
}
catch {
  $Host.UI.WriteErrorLine("`nFailed to install whalebrew...`n")
  Write-Error $error[1]
}
finally {
  $ErrorActionPreference = $orgErrorActionPreference
}
