# Lukaut Logo Assets

## Source Files (SVG)

| File | Description |
|------|-------------|
| `lukaut-logo-primary.svg` | Primary horizontal lockup (light backgrounds) |
| `lukaut-logo-dark.svg` | Dark background variant |
| `lukaut-logo-single-color.svg` | Single-color navy version (reports, fax) |
| `lukaut-logomark.svg` | Standalone mark only |
| `lukaut-favicon.svg` | Optimized favicon source |
| `lukaut-app-icon.svg` | App icon with rounded background |
| `lukaut-og-image.svg` | Open Graph social sharing image |

## Converting to PNG

### Using Inkscape (recommended)
```bash
# Favicon 16x16
inkscape lukaut-favicon.svg -w 16 -h 16 -o favicon-16.png

# Favicon 32x32
inkscape lukaut-favicon.svg -w 32 -h 32 -o favicon-32.png

# App icon 512x512
inkscape lukaut-app-icon.svg -w 512 -h 512 -o app-icon-512.png

# Open Graph image
inkscape lukaut-og-image.svg -w 1200 -h 630 -o og-image.png
```

### Using rsvg-convert
```bash
# Favicon
rsvg-convert -w 16 -h 16 lukaut-favicon.svg > favicon-16.png
rsvg-convert -w 32 -h 32 lukaut-favicon.svg > favicon-32.png

# App icon
rsvg-convert -w 512 -h 512 lukaut-app-icon.svg > app-icon-512.png

# Open Graph
rsvg-convert -w 1200 -h 630 lukaut-og-image.svg > og-image.png
```

### Using Figma/Sketch
Import the SVG and export at the desired dimensions.

## Font Note

The wordmark uses **Satoshi Bold (700)** from Fontshare. The text has been converted to paths in all SVGs, so no font installation is required for the logos to render correctly.

For web usage, include Satoshi via:
```html
<link href="https://api.fontshare.com/v2/css?f[]=satoshi@700&display=swap" rel="stylesheet">
```

## Color Reference

| Color | Hex | Usage |
|-------|-----|-------|
| Slate Navy | `#1E3A5F` | Primary brand, wordmark |
| Safety Orange | `#FF6B35` | Accent, front L layer |
| Clean White | `#FEFEFE` | Light text, backgrounds |
