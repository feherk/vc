#!/bin/bash
set -euo pipefail

# Build a macOS DMG installer for VC
# Usage: ./scripts/build-dmg.sh [version]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION="${1:-$(grep 'var Version' "$PROJECT_DIR/main.go" | sed 's/.*"\(.*\)".*/\1/')}"
APP_NAME="VC"
BUNDLE_ID="com.feherkaroly.vc"
ICON_PNG="$PROJECT_DIR/assets/icon.png"
BUILD_DIR="$PROJECT_DIR/build"
APP_BUNDLE="$BUILD_DIR/$APP_NAME.app"
DMG_NAME="VC-${VERSION}-macOS"
DMG_PATH="$PROJECT_DIR/dist/$DMG_NAME.dmg"

# Code signing
SIGN_IDENTITY="Developer ID Application: Károly Fehér (YG66KQ8KDT)"
NOTARIZE_PROFILE="vc-notarize"

echo "==> Building VC v${VERSION} DMG installer"

# Clean previous build
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
mkdir -p "$PROJECT_DIR/dist"

# ── 1. Build universal binary (arm64 + amd64) ──────────────────────────
echo "==> Compiling arm64..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.Version=${VERSION}" \
    -o "$BUILD_DIR/vc-arm64" "$PROJECT_DIR"

echo "==> Compiling amd64..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.Version=${VERSION}" \
    -o "$BUILD_DIR/vc-amd64" "$PROJECT_DIR"

echo "==> Creating universal binary..."
lipo -create -output "$BUILD_DIR/vc" "$BUILD_DIR/vc-arm64" "$BUILD_DIR/vc-amd64"
rm "$BUILD_DIR/vc-arm64" "$BUILD_DIR/vc-amd64"

# ── 2. Convert icon PNG → icns ──────────────────────────────────────────
ICNS_PATH=""
if [ -f "$ICON_PNG" ]; then
    echo "==> Converting icon..."
    ICONSET="$BUILD_DIR/AppIcon.iconset"
    mkdir -p "$ICONSET"

    sips -z   16   16 "$ICON_PNG" --out "$ICONSET/icon_16x16.png"      >/dev/null
    sips -z   32   32 "$ICON_PNG" --out "$ICONSET/icon_16x16@2x.png"   >/dev/null
    sips -z   32   32 "$ICON_PNG" --out "$ICONSET/icon_32x32.png"      >/dev/null
    sips -z   64   64 "$ICON_PNG" --out "$ICONSET/icon_32x32@2x.png"   >/dev/null
    sips -z  128  128 "$ICON_PNG" --out "$ICONSET/icon_128x128.png"    >/dev/null
    sips -z  256  256 "$ICON_PNG" --out "$ICONSET/icon_128x128@2x.png" >/dev/null
    sips -z  256  256 "$ICON_PNG" --out "$ICONSET/icon_256x256.png"    >/dev/null
    sips -z  512  512 "$ICON_PNG" --out "$ICONSET/icon_256x256@2x.png" >/dev/null
    sips -z  512  512 "$ICON_PNG" --out "$ICONSET/icon_512x512.png"    >/dev/null
    sips -z 1024 1024 "$ICON_PNG" --out "$ICONSET/icon_512x512@2x.png" >/dev/null

    iconutil -c icns "$ICONSET" -o "$BUILD_DIR/AppIcon.icns"
    ICNS_PATH="$BUILD_DIR/AppIcon.icns"
    rm -rf "$ICONSET"
else
    echo "    (no icon at assets/icon.png — skipping)"
fi

# ── 3. Create .app bundle ───────────────────────────────────────────────
echo "==> Creating app bundle..."
mkdir -p "$APP_BUNDLE/Contents/MacOS"
mkdir -p "$APP_BUNDLE/Contents/Resources"

# Copy binary
cp "$BUILD_DIR/vc" "$APP_BUNDLE/Contents/Resources/vc"
chmod +x "$APP_BUNDLE/Contents/Resources/vc"

# Copy icon
if [ -n "$ICNS_PATH" ]; then
    cp "$ICNS_PATH" "$APP_BUNDLE/Contents/Resources/AppIcon.icns"
fi

# Create launcher script (opens Terminal with vc)
cat > "$APP_BUNDLE/Contents/MacOS/VC" << 'LAUNCHER'
#!/bin/bash
BINARY="$(dirname "$0")/../Resources/vc"
osascript - "$BINARY" << 'APPLESCRIPT'
on run argv
    set vcBinary to item 1 of argv
    tell application "Terminal"
        activate
        do script "exec " & quoted form of vcBinary
    end tell
end run
APPLESCRIPT
LAUNCHER
chmod +x "$APP_BUNDLE/Contents/MacOS/VC"

# Info.plist
ICON_ENTRY=""
if [ -n "$ICNS_PATH" ]; then
    ICON_ENTRY="<key>CFBundleIconFile</key>
	<string>AppIcon</string>"
fi

cat > "$APP_BUNDLE/Contents/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>${APP_NAME}</string>
	<key>CFBundleDisplayName</key>
	<string>VC File Manager</string>
	<key>CFBundleIdentifier</key>
	<string>${BUNDLE_ID}</string>
	<key>CFBundleVersion</key>
	<string>${VERSION}</string>
	<key>CFBundleShortVersionString</key>
	<string>${VERSION}</string>
	<key>CFBundleExecutable</key>
	<string>VC</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	${ICON_ENTRY}
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
PLIST

# ── 4. Code sign ────────────────────────────────────────────────────────
echo "==> Signing app bundle..."
codesign --force --options runtime --sign "$SIGN_IDENTITY" "$APP_BUNDLE/Contents/Resources/vc"
codesign --force --options runtime --sign "$SIGN_IDENTITY" "$APP_BUNDLE"

echo "==> Verifying signature..."
codesign --verify --verbose "$APP_BUNDLE"

# ── 5. Create DMG ───────────────────────────────────────────────────────
echo "==> Creating DMG..."

DMG_STAGING="$BUILD_DIR/dmg-staging"
mkdir -p "$DMG_STAGING"
cp -R "$APP_BUNDLE" "$DMG_STAGING/"
ln -s /Applications "$DMG_STAGING/Applications"

# Remove old DMG if exists
rm -f "$DMG_PATH"

# Create read-write DMG first, then set volume icon, then convert to read-only
DMG_RW="$BUILD_DIR/$DMG_NAME-rw.dmg"

hdiutil create -volname "$APP_NAME" \
    -srcfolder "$DMG_STAGING" \
    -ov -format UDRW \
    "$DMG_RW"

# Set volume icon if we have one
if [ -n "$ICNS_PATH" ]; then
    echo "==> Setting volume icon..."
    MOUNT_DIR=$(hdiutil attach "$DMG_RW" -readwrite -noverify -noautoopen | grep "/Volumes/" | sed 's/.*\/Volumes/\/Volumes/')
    cp "$ICNS_PATH" "$MOUNT_DIR/.VolumeIcon.icns"
    SetFile -a C "$MOUNT_DIR"
    hdiutil detach "$MOUNT_DIR" -quiet
fi

# Convert to compressed read-only
hdiutil convert "$DMG_RW" -format UDZO -o "$DMG_PATH"
rm -f "$DMG_RW"

# Sign the DMG itself
echo "==> Signing DMG..."
codesign --force --sign "$SIGN_IDENTITY" "$DMG_PATH"

# ── 6. Notarize ─────────────────────────────────────────────────────────
echo "==> Submitting for notarization (this may take a few minutes)..."
xcrun notarytool submit "$DMG_PATH" --keychain-profile "$NOTARIZE_PROFILE" --wait

echo "==> Stapling notarization ticket..."
xcrun stapler staple "$DMG_PATH"

# ── 7. Clean up ─────────────────────────────────────────────────────────
rm -rf "$BUILD_DIR"

echo ""
echo "==> Done! Signed & notarized DMG created at:"
echo "    dist/$DMG_NAME.dmg"
echo ""
echo "    Size: $(du -h "$DMG_PATH" | cut -f1)"
