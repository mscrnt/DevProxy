# PowerShell script to create a simple icon for DevProxy
Add-Type -AssemblyName System.Drawing

$width = 32
$height = 32

# Create a new bitmap
$bitmap = New-Object System.Drawing.Bitmap($width, $height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)

# Set background
$graphics.Clear([System.Drawing.Color]::FromArgb(0, 50, 100, 200))

# Draw a shield-like shape
$brush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::White)
$pen = New-Object System.Drawing.Pen([System.Drawing.Color]::White, 2)

# Draw "DP" text
$font = New-Object System.Drawing.Font("Arial", 14, [System.Drawing.FontStyle]::Bold)
$stringFormat = New-Object System.Drawing.StringFormat
$stringFormat.Alignment = [System.Drawing.StringAlignment]::Center
$stringFormat.LineAlignment = [System.Drawing.StringAlignment]::Center

$rect = New-Object System.Drawing.RectangleF(0, 0, $width, $height)
$graphics.DrawString("DP", $font, $brush, $rect, $stringFormat)

# Save as ICO
$iconPath = Join-Path (Split-Path $PSScriptRoot) "cmd\devproxy-tray\icon.ico"

# Create icon
$memoryStream = New-Object System.IO.MemoryStream
$bitmap.Save($memoryStream, [System.Drawing.Imaging.ImageFormat]::Png)
$bytes = $memoryStream.ToArray()

# Write ICO header
$iconStream = New-Object System.IO.FileStream($iconPath, [System.IO.FileMode]::Create)
$writer = New-Object System.IO.BinaryWriter($iconStream)

# ICO header
$writer.Write([uint16]0)        # Reserved
$writer.Write([uint16]1)        # Type (1 for icon)
$writer.Write([uint16]1)        # Number of images

# Image directory
$writer.Write([byte]$width)     # Width
$writer.Write([byte]$height)    # Height
$writer.Write([byte]0)          # Color palette
$writer.Write([byte]0)          # Reserved
$writer.Write([uint16]1)        # Color planes
$writer.Write([uint16]32)       # Bits per pixel
$writer.Write([uint32]$bytes.Length) # Size of image data
$writer.Write([uint32]22)       # Offset to image data

# Write PNG data
$writer.Write($bytes)

$writer.Close()
$iconStream.Close()

# Cleanup
$graphics.Dispose()
$bitmap.Dispose()

Write-Host "Icon created at: $iconPath"