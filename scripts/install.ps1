$ErrorActionPreference = 'Stop'

$repository = 'machbase/neo-mcp'
switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    'X64' { $architecture = 'amd64' }
    'Arm64' { $architecture = 'arm64' }
    default { throw "Unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
}

$archive = "neo-mcp_windows_$architecture.zip"
$release = if ($env:NEO_MCP_VERSION) { $env:NEO_MCP_VERSION } else { 'latest' }
$releasePath = if ($release -eq 'latest') { 'releases/latest/download' } else { "releases/download/$release" }
$url = "https://github.com/$repository/$releasePath/$archive"
$destination = if ($env:NEO_MCP_INSTALL_DIR) { $env:NEO_MCP_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'neo-mcp\bin' }
$temporary = Join-Path ([System.IO.Path]::GetTempPath()) ("neo-mcp-" + [System.Guid]::NewGuid())

try {
    New-Item -ItemType Directory -Force -Path $temporary, $destination | Out-Null
    $archivePath = Join-Path $temporary $archive
    Invoke-WebRequest -Uri $url -OutFile $archivePath
    Expand-Archive -Path $archivePath -DestinationPath $temporary
    Copy-Item -Force (Join-Path $temporary "neo-mcp_windows_$architecture\neo-mcp.exe") (Join-Path $destination 'neo-mcp.exe')
    Write-Output "Installed neo-mcp to $(Join-Path $destination 'neo-mcp.exe')"
}
finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $temporary
}
