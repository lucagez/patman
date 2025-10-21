# Patman installer script for PowerShell
# Usage: Invoke-Expression (Invoke-WebRequest -Uri "https://raw.githubusercontent.com/lucagez/patman/main/install.ps1" -UseBasicParsing).Content

function Get-PatmanOS {
    # Windows is the only supported OS for this script
    return "Windows"
}

function Get-PatmanArch {
    switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
        "X64" { return "x86_64" }
        "Arm64" { return "arm64" }
        # Add other architectures if needed
        default {
            Write-Error "Unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)"
            exit 1
        }
    }
}

function Get-LatestPatmanVersion {
    try {
        $response = Invoke-WebRequest -Uri "https://api.github.com/repos/lucagez/patman/releases/latest" -UseBasicParsing
        $json = ConvertFrom-Json $response.Content
        return $json.tag_name
    } catch {
        Write-Error "Could not determine latest version: $_"
        exit 1
    }
}

function Install-Patman {
    $OS = Get-PatmanOS
    $ARCH = Get-PatmanArch
    $VERSION = $env:VERSION
    if ([string]::IsNullOrEmpty($VERSION)) {
        $VERSION = Get-LatestPatmanVersion
    }

    if ([string]::IsNullOrEmpty($VERSION)) {
        Write-Error "Error: Could not determine latest version"
        exit 1
    }

    Write-Host "Installing patman ${VERSION} for ${OS}_${ARCH}..."

    $BINARY_NAME = "patman_${OS}_${ARCH}"
    $ARCHIVE_NAME = "${BINARY_NAME}.zip" # Windows uses zip
    $DOWNLOAD_URL = "https://github.com/lucagez/patman/releases/download/${VERSION}/${ARCHIVE_NAME}"

    $TMP_DIR = Join-Path $env:TEMP "patman_install_$(Get-Random)"
    New-Item -ItemType Directory -Path $TMP_DIR | Out-Null
    $script:cleanupTempDir = $TMP_DIR # Store for finally block or error handling

    Write-Host "Downloading from ${DOWNLOAD_URL}..."
    try {
        Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile (Join-Path $TMP_DIR $ARCHIVE_NAME) -UseBasicParsing
    } catch {
        Write-Error "Error downloading Patman: $_"
        exit 1
    }

    Set-Location $TMP_DIR
    Expand-Archive -Path $ARCHIVE_NAME -DestinationPath $TMP_DIR -Force

    # Determine installation directory
    # Common user-specific path, equivalent to $HOME/.local/bin
    $INSTALL_DIR = Join-Path $env:LOCALAPPDATA "Programs\patman"
    if (-not (Test-Path $INSTALL_DIR)) {
        New-Item -ItemType Directory -Path $INSTALL_DIR | Out-Null
    }

    Write-Host "Installing patman to ${INSTALL_DIR}..."
    Move-Item -Path (Join-Path $TMP_DIR "patman.exe") -Destination (Join-Path $INSTALL_DIR "patman.exe") -Force

    # Add to user PATH if not already present
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$INSTALL_DIR*") {
        Write-Host "Adding $INSTALL_DIR to your user PATH environment variable."
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$INSTALL_DIR", "User")
        Write-Host "You may need to restart your terminal or log out/in for the PATH change to take effect."
    }

    Write-Host "Successfully installed patman to ${INSTALL_DIR}\patman.exe"
    Write-Host ""
    Write-Host "Run 'patman --help' to get started!"

}

# Main execution block with cleanup
try {
    Install-Patman
} finally {
    if (Test-Path $script:cleanupTempDir) {
        Remove-Item -Path $script:cleanupTempDir -Recurse -Force
    }
}
