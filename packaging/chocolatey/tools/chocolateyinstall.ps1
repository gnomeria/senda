$ErrorActionPreference = 'Stop'

# Bumped per release by packaging/update.sh.
$version  = '0.1.4'
$checksum = 'f7493570cee4d54a6c7062ad713cd330454e1b229dc361bdc7b47a0924f8cbb8'

$packageName = 'senda'
$toolsDir    = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url64       = "https://github.com/this-senda/senda/releases/download/v$version/senda_${version}_windows-amd64.zip"

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  checksum64     = $checksum
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# Choco auto-shims every .exe under the tools dir, so both `senda` and
# `senda-desktop` become available on PATH after install.
