# iOS App

GoRead2 ships a native iOS/iPadOS client, a SwiftUI app in `ios/` that consumes the same REST API as the web frontend. The Xcode project is `ios/GoRead2.xcodeproj`, the bundle identifier is `org.jeffreypratt.goread2`, and the deployment target is iOS 16.

This guide covers local development and testing on a physical device. Release distribution through TestFlight is automated; see the [iOS Release Pipeline](deployment.md#ios-release-pipeline-githubworkflowsios-releaseyml) section of the deployment guide. The mobile OAuth handoff is described in the [authentication guide](authentication.md#mobile-client-flow-ios).

## Table of Contents

- [Schemes and the API Base URL](#schemes-and-the-api-base-url)
- [Building and Running in the Simulator](#building-and-running-in-the-simulator)
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

## Building and Running in the Simulator

Open `ios/GoRead2.xcodeproj` in Xcode, select the `GoRead2` scheme and a simulator destination, and run. From the command line:

```bash
xcodebuild -project ios/GoRead2.xcodeproj -scheme GoRead2 \
  -destination 'platform=iOS Simulator,name=iPhone 17' \
  CODE_SIGNING_ALLOWED=NO build
```

Simulator builds need no signing identity or Apple account.

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
