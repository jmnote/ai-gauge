param(
    [string]$Executable = "",
    [string]$OutputDirectory = "",
    [int]$WaitSeconds = 15,
    [int]$RenderWaitSeconds = 20,
    [int]$ThemeSwitchWaitMilliseconds = 300
)

$ErrorActionPreference = "Stop"
$repo = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($Executable)) {
    $binExe = Join-Path $repo "dist\bin\ai-gauge.exe"
    $rootExe = Join-Path $repo "ai-gauge.exe"
    $Executable = if (Test-Path -LiteralPath $binExe -PathType Leaf) { $binExe } else { $rootExe }
}
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) { $OutputDirectory = Join-Path $repo "docs\screenshots" }
if (-not (Test-Path -LiteralPath $Executable -PathType Leaf)) { throw "Executable not found: $Executable" }

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null

Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -ReferencedAssemblies System.Drawing @'
using System;
using System.Drawing;
using System.Drawing.Imaging;
using System.Runtime.InteropServices;

public static class WindowCapture {
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int Left; public int Top; public int Right; public int Bottom; }

    [DllImport("user32.dll")]
    public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);

    [DllImport("user32.dll")]
    public static extern bool PrintWindow(IntPtr hWnd, IntPtr hdcBlt, uint flags);

    [DllImport("user32.dll")]
    public static extern bool SetProcessDPIAware();

    private delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);

    [DllImport("user32.dll")]
    private static extern bool EnumWindows(EnumWindowsProc callback, IntPtr lParam);

    [DllImport("user32.dll")]
    private static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);

    [DllImport("user32.dll")]
    private static extern bool IsWindowVisible(IntPtr hWnd);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    private static extern int GetWindowText(IntPtr hWnd, System.Text.StringBuilder text, int maxCount);

    // FindVisible returns the visible top-level window of processId titled
    // title, or IntPtr.Zero.
    public static IntPtr FindVisible(int processId, string title) {
        IntPtr found = IntPtr.Zero;
        EnumWindows((hWnd, _) => {
            uint pid;
            GetWindowThreadProcessId(hWnd, out pid);
            if (pid != processId || !IsWindowVisible(hWnd)) return true;
            var text = new System.Text.StringBuilder(256);
            GetWindowText(hWnd, text, text.Capacity);
            if (text.ToString() != title) return true;
            found = hWnd;
            return false;
        }, IntPtr.Zero);
        return found;
    }

    public static void Capture(IntPtr handle, string path) {
        SetProcessDPIAware();
        RECT rect;
        if (!GetWindowRect(handle, out rect)) throw new InvalidOperationException("Could not get window bounds.");
        int width = rect.Right - rect.Left;
        int height = rect.Bottom - rect.Top;
        if (width <= 0 || height <= 0) throw new InvalidOperationException("Window has invalid bounds.");

        using (var bitmap = new Bitmap(width, height, PixelFormat.Format32bppArgb)) {
            using (var graphics = Graphics.FromImage(bitmap)) {
                IntPtr hdc = graphics.GetHdc();
                try {
                    if (!PrintWindow(handle, hdc, 2)) throw new InvalidOperationException("PrintWindow failed.");
                } finally { graphics.ReleaseHdc(hdc); }
            }
            bitmap.Save(path, ImageFormat.Png);
        }
    }
}
'@

function New-RoundedRectanglePath {
    param([System.Drawing.Rectangle]$Rectangle, [int]$Radius)
    $diameter = $Radius * 2
    $path = New-Object System.Drawing.Drawing2D.GraphicsPath
    $path.AddArc($Rectangle.X, $Rectangle.Y, $diameter, $diameter, 180, 90)
    $path.AddArc($Rectangle.Right - $diameter, $Rectangle.Y, $diameter, $diameter, 270, 90)
    $path.AddArc($Rectangle.Right - $diameter, $Rectangle.Bottom - $diameter, $diameter, $diameter, 0, 90)
    $path.AddArc($Rectangle.X, $Rectangle.Bottom - $diameter, $diameter, $diameter, 90, 90)
    $path.CloseFigure()
    return $path
}

function Add-RoundedShadow {
    param([string]$SourcePath, [string]$DestinationPath, [int]$Radius = 5, [int]$Padding = 24)
    $source = [System.Drawing.Image]::FromFile($SourcePath)
    try {
        $canvas = New-Object System.Drawing.Bitmap(
            ($source.Width + ($Padding * 2)),
            ($source.Height + ($Padding * 2)),
            [System.Drawing.Imaging.PixelFormat]::Format32bppArgb
        )
        try {
            $graphics = [System.Drawing.Graphics]::FromImage($canvas)
            try {
                $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
                $graphics.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceOver
                $windowRect = [System.Drawing.Rectangle]::new($Padding, $Padding, $source.Width, $source.Height)

                for ($offset = 10; $offset -ge 2; $offset -= 2) {
                    $shadowRect = [System.Drawing.Rectangle]::new(
                        ($Padding - [int]($offset / 2)),
                        ($Padding + $offset),
                        $source.Width + $offset,
                        $source.Height + $offset
                    )
                    $path = New-RoundedRectanglePath -Rectangle $shadowRect -Radius ($Radius + [int]($offset / 3))
                    $brush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(10, 0, 0, 0))
                    try { $graphics.FillPath($brush, $path) } finally { $brush.Dispose(); $path.Dispose() }
                }

                $clip = New-RoundedRectanglePath -Rectangle $windowRect -Radius $Radius
                try {
                    $graphics.SetClip($clip)
                    $graphics.DrawImage($source, $windowRect)
                } finally { $clip.Dispose() }
            } finally { $graphics.Dispose() }
            $canvas.Save($DestinationPath, [System.Drawing.Imaging.ImageFormat]::Png)
        } finally { $canvas.Dispose() }
    } finally { $source.Dispose() }
}

function Wait-Window([System.Diagnostics.Process]$Process, [string]$Title) {
    $deadline = (Get-Date).AddSeconds($WaitSeconds)
    do {
        Start-Sleep -Milliseconds 250
        $handle = [WindowCapture]::FindVisible($Process.Id, $Title)
    } while ($handle -eq 0 -and (Get-Date) -lt $deadline)
    if ($handle -eq 0) { throw "Window '$Title' was not found within $WaitSeconds seconds." }
    return $handle
}

function Save-WindowCapture([IntPtr]$Handle, [string]$Name) {
    $outputPath = Join-Path $OutputDirectory $Name
    $rawPath = Join-Path $env:TEMP ("ai-gauge-window-" + [System.IO.Path]::GetRandomFileName())
    try {
        [WindowCapture]::Capture($Handle, $rawPath)
        Add-RoundedShadow -SourcePath $rawPath -DestinationPath $outputPath
    } finally {
        if (Test-Path -LiteralPath $rawPath) { [System.IO.File]::Delete($rawPath) }
    }
    Write-Output "Created: $outputPath"
}

# Invoke-ThemeButton presses a Settings theme button (aria-label Light/Dark),
# which also saves that theme as the user's preference. aria-pressed makes it
# a UIA toggle button, so it exposes TogglePattern rather than InvokePattern.
# Toggling also focuses the button, so focus then moves to the page itself to
# keep the button's focus ring out of the capture.
function Invoke-ThemeButton([IntPtr]$SettingsHandle, [string]$Name) {
    $root = [System.Windows.Automation.AutomationElement]::FromHandle($SettingsHandle)
    $condition = New-Object System.Windows.Automation.AndCondition(
        (New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ControlTypeProperty, [System.Windows.Automation.ControlType]::Button)),
        (New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty, $Name))
    )
    $deadline = (Get-Date).AddSeconds($WaitSeconds)
    do {
        $button = $root.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $condition)
        if ($null -eq $button) { Start-Sleep -Milliseconds 250 }
    } while ($null -eq $button -and (Get-Date) -lt $deadline)
    if ($null -eq $button) { throw "Settings '$Name' theme button was not found within $WaitSeconds seconds." }
    $button.GetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern).Toggle()
    $document = $root.FindFirst([System.Windows.Automation.TreeScope]::Descendants,
        (New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ControlTypeProperty, [System.Windows.Automation.ControlType]::Document)))
    if ($null -ne $document) { $document.SetFocus() }
}

$process = Start-Process -FilePath $Executable -ArgumentList @("--settings") -PassThru
try {
    $mainHandle = Wait-Window $process "AI Gauge"
    $settingsHandle = Wait-Window $process "AI Gauge - Settings"
    Start-Sleep -Seconds $RenderWaitSeconds
    foreach ($theme in @("light", "dark")) {
        Invoke-ThemeButton $settingsHandle ((Get-Culture).TextInfo.ToTitleCase($theme))
        Start-Sleep -Milliseconds $ThemeSwitchWaitMilliseconds
        Save-WindowCapture $mainHandle "aigauge-main-$theme.png"
        Save-WindowCapture $settingsHandle "aigauge-settings-$theme.png"
    }
} finally {
    if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force }
}
