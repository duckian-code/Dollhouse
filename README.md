# Dollhouse

Dollhouse is a React Native mood-tracking application built with Expo and TypeScript. This
repository currently contains the mobile application foundation for Trello ticket 20.

## Prerequisites

- Node.js 22 LTS or newer
- npm 10 or newer
- An Expo development build for device authentication tests
- Android Studio for local Android development builds
- Xcode on macOS for local iOS development builds

## Setup

```bash
npm install
cp .env.example .env
npm start
```

The example environment targets the shared deployed development backend. The Expo development
server will show options for opening the app on Android, iOS, web, or a physical device.

AWS Amplify authentication depends on native modules that are unavailable in Expo Go. Test login
in the web build or install an Expo development build on the device.

## Environment variables

| Variable                                  | Purpose                       |
| ----------------------------------------- | ----------------------------- |
| `EXPO_PUBLIC_API_URL`                     | Base URL for the backend API  |
| `EXPO_PUBLIC_COGNITO_USER_POOL_ID`        | Cognito user pool identifier  |
| `EXPO_PUBLIC_COGNITO_USER_POOL_CLIENT_ID` | Cognito application client ID |

Variables prefixed with `EXPO_PUBLIC_` are included in the client application bundle. Never use
them for passwords, private API keys, signing keys, or other secrets. Local `.env` files are
ignored by Git; `.env.example` documents the shared contract.

## Commands

```bash
npm start          # Start the Expo development server
npm run android    # Generate and run a local Android development build
npm run android:dev # Generate and run a local Android development build
npm run ios        # Generate and run a local iOS development build
npm run ios:dev    # Generate and run a local iOS development build (macOS)
npm run web        # Open the web build
npm run typecheck  # Check TypeScript
npm run lint       # Check ESLint rules
npm run lint:fix   # Apply safe ESLint fixes
npm run format     # Format tracked source/configuration files
npm run format:check # Check formatting without changing files
npm run doctor     # Run Expo project and native-tooling diagnostics
```

For an EAS development-client build, authenticate with Expo and run:

```bash
npx eas-cli build --profile development --platform android
```

Development-client builds load JavaScript from Metro. Start Metro with `npm start` before opening
the installed development app, then ensure the device can reach the computer over the local
network. To create a standalone internal APK with the JavaScript bundle included, run:

```bash
npx eas-cli build --profile preview --platform android
```

The committed `eas.json` contains the shared development-build profile. Expo account and project
linking are intentionally performed by a team member so the project is created under the team's
Expo organization rather than an individual developer's account.

## Source structure

```text
src/
├── app/          Expo Router screens and layouts
├── components/   Reusable user-interface components
├── config/       Application and environment configuration
├── services/     Backend and platform integration boundaries
└── types/        Shared TypeScript declarations and domain types
```

Feature-specific folders should be added only when their first feature is implemented.
