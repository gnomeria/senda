$ErrorActionPreference = 'Stop'

# The zip is unpacked under the package tools dir and choco removes it along
# with the auto-generated shims on uninstall. Remove any leftover binaries
# defensively in case a shim was left behind.
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
Get-ChildItem -Path $toolsDir -Filter '*.exe' -ErrorAction SilentlyContinue | ForEach-Object {
  Uninstall-BinFile -Name $_.BaseName -ErrorAction SilentlyContinue
}
