# iOS and macOS Apps

GoRead2 ships a native client for iOS, iPadOS, and macOS, a SwiftUI app in `ios/` that consumes the same REST API as the web frontend. The Xcode project is `ios/GoRead2.xcodeproj`, the bundle identifier is `org.jeffreypratt.goread2`, and the deployment targets are iOS 16 and macOS 13.

One target, `GoRead2`, builds for every platform. `SDKROOT` is `auto` and `SUPPORTED_PLATFORMS` lists `iphoneos iphonesimulator macosx`, so the SwiftUI views, `NetworkClient`, and models compile once for all three platforms with no duplicated sources or per-platform target membership. The project uses a filesystem-synchronized root group, so new files under `ios/GoRead2/` join the build automatically.

This guide covers local development and testing on a physical device. Release distribution through TestFlight is automated; see the [iOS Release Pipeline](deployment.md#ios-release-pipeline-githubworkflowsios-releaseyml) section of the deployment guide. The native OAuth handoff is described in the [authentication guide](authentication.md#native-client-flow-ios-and-macos).

## Table of Contents

- [Schemes and the API Base URL](#schemes-and-the-api-base-url)
- [App Icon](#app-icon)
- [Building and Running in the Simulator](#building-and-running-in-the-simulator)
- [Building for macOS](#building-for-macos)
- [Platform Differences](#platform-differences)
- [Running on a Physical Device](#running-on-a-physical-device)
- [Free-Account Limitations](#free-account-limitations)
- [TestFlight](#testflight)

## Schemes and the API Base URL

The backend the app talks to is a build setting, `API_BASE_URL`, exposed to the app through `Info.plist`:

| Scheme | Configuration | `API_BASE_URL` |
|--------|---------------|----------------|
| `GoRead2` | Debug | `http://localhost:8080` |
| `GoRead2-Release` | Release | `https://goreadapp.com` |

The Debug value suits the simulator, where localhost is the Mac running `make dev`. On a physical device localhost is the device itself, so the Debug scheme cannot reach a local backend without changes. When running on a device, either:

- **Use the `GoRead2-Release` scheme** to talk to production. This is the simplest option and exercises the real OAuth flow.
- **Point Debug at the Mac's LAN address**: change the Debug `API_BASE_URL` in the project's build settings to `http://<mac-lan-ip>:8080` and start the dev server with `make dev`. `Info.plist` already sets `NSAllowsLocalNetworking`, so App Transport Security permits the plain-HTTP connection on the local network.

## App Icon

The app icon lives in `ios/GoRead2/Assets.xcassets/AppIcon.appiconset/AppIcon.png`, a single 1024x1024 opaque PNG that Xcode scales for every context. Its source of truth is `ios/AppIcon.svg`, which adapts the web favicon's branding, a white RSS glyph with reading lines on a blue gradient. After editing the SVG, regenerate the PNG on macOS with:

```bash
qlmanage -t -s 1024 -o /tmp ios/AppIcon.svg
magick /tmp/AppIcon.svg.png -alpha remove -alpha off \
  PNG24:ios/GoRead2/Assets.xcassets/AppIcon.appiconset/AppIcon.png
```

`qlmanage` renders the SVG through WebKit, which handles gradients and arc paths that ImageMagick's built-in SVG renderer does not. The `-alpha` flags strip the alpha channel, which App Store Connect rejects in app icons.

## Building and Running in the Simulator

Open `ios/GoRead2.xcodeproj` in Xcode, select the `GoRead2` scheme and a simulator destination, and run. From the command line:

```bash
xcodebuild -project ios/GoRead2.xcodeproj -scheme GoRead2 \
  -destination 'platform=iOS Simulator,name=iPhone 17' \
  CODE_SIGNING_ALLOWED=NO build
```

Simulator builds need no signing identity or Apple account.

## Building for macOS

Select the `My Mac` destination in Xcode and run, or from the command line:

```bash
xcodebuild -project ios/GoRead2.xcodeproj -scheme GoRead2 \
  -destination 'platform=macOS' build
```

The Mac build is a native AppKit-backed SwiftUI app, not Mac Catalyst and not the iPad build running on Apple silicon: `SUPPORTS_MACCATALYST` and `SUPPORTS_MAC_DESIGNED_FOR_IPHONE_IPAD` are both `NO`.

`ios/GoRead2/GoRead2-macOS.entitlements` applies to the macOS SDK only and enables the App Sandbox, outgoing network access, and read/write access to user-selected files for OPML import and export. The hardened runtime is on for both configurations. The shared `Info.plist` registers the `goread2://` URL scheme for the OAuth callback on every platform, and the UIKit-only `INFOPLIST_KEY_UI*` settings are scoped to the iOS SDKs so they stay out of the Mac bundle.

## Platform Differences

Shared views carry an `#if os(...)` branch only where platform behaviour genuinely differs. `ios/GoRead2/Components/PlatformCompat.swift` holds the shims for modifiers that exist on one platform, so call sites read the same everywhere.

| Behaviour | iOS and iPadOS | macOS |
|-----------|----------------|-------|
| Root layout | Three-pane split on iPad, navigation stack on iPhone | Always the three-pane split |
| External links | In-app `SFSafariViewController` sheet | Default browser |
| OPML export | Share sheet | Revealed in the Finder |
| Article web view | `UIViewRepresentable`, with swipe gestures between articles | `NSViewRepresentable`, no swipe gestures |
| OAuth anchor | Key window of the active `UIWindowScene` | `NSApplication.shared.keyWindow` |
| `?client=` value | `ios` | `macos` |

## Running on a Physical Device

Installing on a device requires a signing identity, but not a paid Apple Developer Program membership: a free Apple ID provides a "Personal Team" that can sign builds for personal devices.

### One-time Xcode setup

1. In Xcode → Settings → Accounts, add an Apple ID. A free account gets a Personal Team automatically.
2. Open the project, select the GoRead2 target → Signing & Capabilities, and pick the team from the Team dropdown. The project already uses automatic signing, so Xcode creates the development certificate and provisioning profile itself.

### One-time device setup

1. **Connect and pair**: attach the device over USB and choose it as the run destination. Xcode prompts to pair; confirm on the device. After the first pairing, Xcode can target the device over Wi-Fi (Window → Devices and Simulators → "Connect via network").
2. **Enable Developer Mode** (iOS 16 and later): on the device, Settings → Privacy & Security → Developer Mode, toggle it on, and restart the device. The toggle only appears after the device has seen Xcode at least once.
3. **Run** from Xcode. The first install fails to launch until the signing certificate is trusted on the device: Settings → General → VPN & Device Management, select the developer app entry, and tap Trust.

Subsequent runs need none of this; select the device and run.

## Free-Account Limitations

Personal Team signing carries restrictions that paid memberships do not:

- **Installs expire after 7 days.** The app icon remains but the app refuses to launch; re-running from Xcode re-signs it and restarts the clock.
- At most 3 sideloaded apps on a device at once, and at most 10 unique bundle IDs registered per week.
- No TestFlight and no App Store distribution.
- No entitlement-gated capabilities such as push notifications. GoRead2 currently uses none, so this does not affect the app.

## TestFlight

Distributing builds to testers' devices without a cable requires the paid Apple Developer Program and goes through TestFlight. The CI pipeline builds and uploads automatically on pushes to `main`; the pipeline, its one-time Apple setup, and the versioning scheme are documented in the [deployment guide](deployment.md#ios-release-pipeline-githubworkflowsios-releaseyml).
