# Apple Release Guide

Process for releasing the GoRead2 native app on iOS, iPadOS, and macOS. Local development and on-device testing are covered in the [iOS and macOS app guide](ios.md).

## Table of Contents

- [Channels and Status](#channels-and-status)
- [Shared Setup](#shared-setup)
- [iOS and iPadOS](#ios-and-ipados)
- [macOS](#macos)
- [Fastlane Updates](#fastlane-updates)
- [Renewals](#renewals)
- [Troubleshooting](#troubleshooting)

## Channels and Status

All three platforms build from one Xcode target, `GoRead2`, and all require the paid Apple Developer Program. iOS and iPadOS share one build, so a TestFlight upload serves both.

| Platform | Channel | Workflow | Status |
|----------|---------|----------|--------|
| iOS | TestFlight | `.github/workflows/ios-release.yml` | Never run. Disabled, and its signing assets do not exist. |
| iPadOS | TestFlight, same build as iOS | `.github/workflows/ios-release.yml` | Never run, as for iOS. |
| macOS | Notarized DMG on GitHub releases | `.github/workflows/macos-release.yml` | A manual run has produced a notarized DMG. No tagged release has been published. |

The following are not tested or not implemented:

- **iOS and iPadOS**: the `beta` lane has never run, so its signing, build, and upload steps are all unconfirmed.
- **macOS**: the release job's publishing step has never run, because it runs only for a `macos-v*` tag.
- **App Store**: the pipeline stops at TestFlight. Nothing in this repository submits a build to App Review.

## Shared Setup

Both pipelines share an App Store Connect API key, a certificates repository, and the GitHub configuration below. All of this is in place except where the [iOS](#ios-and-ipados) section says otherwise.

### Code signing

Code signing uses [fastlane match](https://docs.fastlane.tools/actions/match/). Certificates and provisioning profiles live encrypted in a private repository, `jeffreyp/goread2-certificates`, and CI fetches them read-only into a temporary keychain created by `setup_ci`. Xcode Cloud's managed signing was rejected because its configuration lives in the App Store Connect UI rather than in this repository.

match stores assets on the repository's `master` branch, which is match's default, while the repository's default branch is `main`. The repository holds one set of assets per channel, kept apart by match's storage layout. Viewing the repository on GitHub shows `main`, which contains only a README.

The Xcode project stays on automatic signing for local development. Each lane switches the runner's working copy to manual signing with `update_code_signing_settings`, and nothing is committed back.

### Certificates repository access

CI authenticates to the certificates repository with a read-only SSH deploy key, which has no expiry date and no access to any other repository. To create or replace it:

```bash
ssh-keygen -t ed25519 -N "" -C "goread2-ci-match" -f match_deploy_key
gh repo deploy-key add match_deploy_key.pub --repo jeffreyp/goread2-certificates --title "goread2 CI (fastlane match)"
gh secret set MATCH_GIT_PRIVATE_KEY --repo jeffreyp/goread2 < match_deploy_key
rm match_deploy_key match_deploy_key.pub
```

The key has no passphrase because CI cannot supply one. The deploy key cannot write, so generating signing assets on a developer machine uses the developer's own GitHub credentials over HTTPS.

### App Store Connect API key

Under Users and Access → Integrations in [App Store Connect](https://appstoreconnect.apple.com/), create a team key with the App Manager role. Record the Key ID and Issuer ID, and download the `.p8` file. macOS needs this key too, despite never reaching the App Store, because match and `xcrun notarytool` authenticate with it.

### GitHub configuration

Repository Settings → Secrets and variables → Actions:

| Name | Kind | Value |
|------|------|-------|
| `APPLE_TEAM_ID` | variable | Apple Developer Team ID. The macOS job skips while it is unset. |
| `IOS_TESTFLIGHT_ENABLED` | variable | Any non-empty value. The iOS job skips while it is unset. |
| `ASC_KEY_ID` | secret | App Store Connect API Key ID |
| `ASC_ISSUER_ID` | secret | App Store Connect API Issuer ID |
| `ASC_KEY_CONTENT` | secret | Base64 of the `.p8` key: `base64 -i AuthKey_XXXXXXXX.p8` |
| `MATCH_GIT_URL` | secret | SSH URL of the certificates repository: `git@github.com:jeffreyp/goread2-certificates.git` |
| `MATCH_GIT_PRIVATE_KEY` | secret | Private key of the deploy key |
| `MATCH_PASSWORD` | secret | match encryption passphrase |

`MATCH_PASSWORD` encrypts every asset in the repository, so all assets must use the same passphrase. GitHub does not reveal a stored secret, so the passphrase must also be kept outside GitHub.

### Local toolchain

The commands that generate signing assets run on a developer machine with the Ruby version in `ios/.ruby-version`. See [Ruby Toolchain for fastlane](ios.md#ruby-toolchain-for-fastlane).

## iOS and iPadOS

Every push to `main` that touches `ios/**` or the workflow file runs `bundle exec fastlane beta` on a GitHub-hosted macOS runner. The `beta` lane fetches signing assets, builds a signed Release IPA with the `GoRead2-Release` scheme, and uploads it to TestFlight through the App Store Connect API. The job skips itself until both `IOS_TESTFLIGHT_ENABLED` and `APPLE_TEAM_ID` are set, and the workflow has no manual trigger.

### First-time setup

1. **Create the app record.** In [App Store Connect](https://appstoreconnect.apple.com/), register the bundle ID `org.jeffreypratt.goread2` and create the app. TestFlight uploads require the record to exist. The App ID must support iOS as well as macOS.
2. **Generate the signing assets** from a developer machine. match prompts for an Apple ID sign-in and the encryption passphrase, which must be the existing `MATCH_PASSWORD`:

   ```bash
   cd ios
   bundle install
   MATCH_GIT_URL=https://github.com/jeffreyp/goread2-certificates.git \
   APPLE_TEAM_ID=<team id> \
   bundle exec fastlane match appstore
   ```

3. **Enable the pipeline.** Once set, this variable causes every later push that touches `ios/**` to upload a build, including Dependabot updates to `ios/Gemfile.lock`:

   ```bash
   gh variable set IOS_TESTFLIGHT_ENABLED --repo jeffreyp/goread2 --body 1
   ```

4. **Trigger the first build** by pushing a commit that touches `ios/**`.
5. **Confirm the upload.** The build appears in App Store Connect → TestFlight once Apple finishes processing it. `Info.plist` does not set `ITSAppUsesNonExemptEncryption`, so App Store Connect asks for an export compliance answer before the build can be tested.
6. **Install it.** Add testers to an internal group in App Store Connect, then install the TestFlight app on an iPhone and on an iPad. Test both, since the layouts differ. See [Platform Differences](ios.md#platform-differences).

### Versioning

- **Build number (`CFBundleVersion`)**: the commit count on `main` (`git rev-list --count HEAD`), injected at build time as `CURRENT_PROJECT_VERSION`. Every push produces a strictly increasing build number without version-bump commits. The workflow checks out full history to make the count available, and serializes runs because TestFlight rejects a build number it has already processed.
- **Marketing version (`CFBundleShortVersionString`)**: manual. Edit `MARKETING_VERSION` in `ios/GoRead2.xcodeproj` when cutting a user-visible release. TestFlight groups builds under it.

### Testers

Internal testers must be App Store Connect users on the team, and their builds need no review. External testers require Beta App Review of the first build of a version.

## macOS

A `macos-v*` tag runs `bundle exec fastlane mac release` on a GitHub-hosted macOS runner, which builds the Mac app, signs it with a Developer ID certificate, notarizes it, and publishes the disk image as a GitHub release. `workflow_dispatch` runs the same lane and leaves the DMG as a workflow artifact, without publishing. The job skips itself until `APPLE_TEAM_ID` is set.

### Developer ID, not the Mac App Store

GoRead2 sells its subscription through Stripe, and the macOS Settings window links out to the Stripe customer portal. App Review guideline 3.1.1 does not permit that arrangement, so a Mac App Store build would mean adding StoreKit in-app purchase for a subscription that already exists on the web. Developer ID signing and notarization keep the existing billing and take App Review off the release path.

Direct distribution has these consequences:

- **Updates are manual.** A new release means downloading a new DMG. An in-app updater is tracked as issue gr-i0zo.
- **No universal purchase.** The Mac app is not bundled with the iOS app on the App Store.
- **Distribution is self-hosted.** The DMG lives on the repository's GitHub releases page.

### Signing and notarization

The certificates repository holds a Developer ID Application certificate and profile under match's `macos` platform. The lane reads the profile name from match's `MATCH_PROVISIONING_PROFILE_MAPPING`, because match derives the macOS profile name itself.

Notarization runs twice through `xcrun notarytool`, using the App Store Connect API key:

1. The exported `GoRead2.app` is notarized and stapled. Stapling before packaging means the ticket travels with the bundle after it is dragged to `/Applications`, so a first launch on a machine with no network still passes Gatekeeper.
2. The DMG is built from the stapled app, signed with the same certificate, then notarized and stapled itself. Gatekeeper checks the disk image separately when it is opened from a quarantined download.

The hardened runtime is enabled for both configurations in `ios/GoRead2.xcodeproj`, which notarization requires. The entitlements are the sandboxed set described in [Building for macOS](ios.md#building-for-macos).

### Regenerating the signing assets

These assets already exist. Regenerate them only if they are lost or expire. Apple limits the number of Developer ID Application certificates per account and they cannot be revoked freely, so let match reuse the existing one where possible. The App ID must support macOS, or match cannot create the profile.

```bash
cd ios
MATCH_GIT_URL=https://github.com/jeffreyp/goread2-certificates.git \
APPLE_TEAM_ID=<team id> \
bundle exec fastlane match developer_id --platform macos
```

### Versioning

- **Build number (`CFBundleVersion`)**: the commit count, as on iOS.
- **Marketing version (`CFBundleShortVersionString`)**: taken from the tag. Pushing `macos-v1.2.0` builds and names version 1.2.0, so a release needs no edit to `MARKETING_VERSION`. A `workflow_dispatch` run falls back to the project's value.

### Testing before a release

1. Run the workflow and download the artifact:

   ```bash
   gh workflow run macos-release.yml --repo jeffreyp/goread2 --ref main
   gh run download <run id> --repo jeffreyp/goread2 --name GoRead2-macOS
   ```

2. Check the notarization ticket and the Gatekeeper assessment of the disk image:

   ```bash
   xcrun stapler validate GoRead2-<version>.dmg
   spctl --assess --type open --context context:primary-signature --verbose GoRead2-<version>.dmg
   ```

3. Open the DMG, drag the app to `/Applications`, launch it, and sign in with Google. The Release build talks to production.

### Cutting a release

```bash
git tag macos-v1.2.0
git push origin macos-v1.2.0
```

The workflow creates a GitHub release named after the tag and attaches the DMG. Rerunning it for the same tag replaces the attached DMG. Afterwards, download the DMG from the releases page and repeat step 3 above, since a browser download is quarantined and exercises the Gatekeeper path that users see.

## Fastlane Updates

Dependabot opens weekly pull requests that update fastlane in `ios/Gemfile.lock`. Merging one does not exercise the macOS `release` lane, which runs only on a `macos-v*` tag or `workflow_dispatch`. A fastlane release that removes or changes an action option therefore fails the next time the lane runs, which can be at release time.

To check an update, run the workflow manually after merging it:

```bash
gh workflow run macos-release.yml --repo jeffreyp/goread2 --ref main
```

This submits the app and disk image to Apple for notarization and leaves the DMG as a workflow artifact, without publishing a release.

## Renewals

| Item | Renewal | Symptom when it lapses |
|------|---------|------------------------|
| Apple Developer Program membership | Annual | Signing, notarization, and uploads stop working |
| Apple agreements | Whenever Apple updates them. The account holder accepts them on the Apple Developer site and in App Store Connect. | Notarization fails with HTTP 403. See [Troubleshooting](#troubleshooting). |
| Apple Distribution certificate (iOS) | Roughly annual. Repeat `fastlane match appstore` from a developer machine. | The match step fails on iOS only |
| Developer ID certificate (macOS) | Multi-year. See [Regenerating the signing assets](#regenerating-the-signing-assets). | The match step fails on macOS only |

The certificates repository's deploy key and the App Store Connect API key have no expiry date.

## Troubleshooting

| Error in the workflow log | Cause | Fix |
|---------------------------|-------|-----|
| `HTTP status code: 403. A required agreement is missing or has expired`, followed by a `JSON::ParserError` | An Apple agreement needs accepting. The parser error is fastlane failing to read the 403 text and is not a separate fault. | The account holder accepts the agreement on the [Apple Developer site](https://developer.apple.com/account) and in App Store Connect. Acceptance can take up to about 30 minutes to reach the notary service. |
| `Could not find option '<name>' in the list of available options` | A fastlane update changed an action's options. | Compare the action in the old and new fastlane versions, and update `ios/fastlane/Fastfile`. |
| Git authentication failure at the match step | The deploy key or `MATCH_GIT_URL` is wrong. | Replace the key as described in [Certificates repository access](#certificates-repository-access), and confirm that the URL is the SSH form. |
| match cannot decrypt the repository | `MATCH_PASSWORD` differs from the passphrase that encrypted the assets. | Set the secret to the correct passphrase. |
| match cannot find a certificate or profile | The assets for that channel have not been generated. | Run the matching `fastlane match` command from a developer machine. CI runs match read-only and cannot create them. |
