# Tachigo Design-Source Naming Conventions

This guide explains how to name Tachigo design-source files, prototype files, final assets, and local workspace folders.

It is written for designers and collaborators who are new to GitHub. The main idea is simple:

- Design-source folders can keep drafts and version history.
- GitHub should only receive clear, stable, final names.
- Local workspace folders are for organization only and should not be committed.

## Scope

This document tracks the cleanup and documentation of Tachigo design-source folder naming conventions.

In scope:

- Organize design-source naming rules.
- Keep runtime extension code unchanged.
- Avoid large code-aware rename work in this PR.

Out of scope:

- Chrome extension runtime changes.
- JavaScript to TypeScript migration.
- Build or dependency changes.

## 1. Core Naming Rule

Use English, lowercase, and kebab-case for engineering-facing files and folders.

Good:

```text
opening-01-loop.mp4
opening-02-loop.mp4
login-character.svg
login-background.svg
character-select-bg.png
crab-say-hi-loop.mp4
crab-mining-loop.mp4
claim-panel.tsx
character-select-screen.tsx
login-screen.tsx
```

Avoid:

```text
Login Character.svg
login_character.svg
螃蟹sayHi最終版.mp4
Screenshot 2026-05-15 at 2.22.57 PM.png
openart-02177936768261700000000000000000000ffffc0a869babb58fb.mp4
final_final_真的最後.mp4
```

Kebab-case is preferred because it is safer for URLs, imports, build tools, Chrome extension paths, and cross-platform file systems.

## 2. Local Workspace Folders

The local workspace on the user's Mac is organized outside the GitHub repo:

```text
/Users/nu/Desktop/tachigo-workspace/
  01_design-source/
  02_github/
    tachigo/
```

Use `01_design-source/` for local design work, references, draft exports, and experiments.

Use `02_github/tachigo/` for the cloned GitHub repo.

Do not commit local workspace folders such as `01_design-source/` or `02_github/` into GitHub.

## 3. Design Source vs GitHub Repo

Design source:

```text
/Users/nu/Desktop/tachigo-workspace/01_design-source/
```

Use this for:

- Figma exports
- OpenArt outputs
- Runway outputs
- video drafts
- screenshots
- references
- large experiments
- archived attempts

GitHub repo:

```text
/Users/nu/Desktop/tachigo-workspace/02_github/tachigo/
```

Use this for:

- final product code
- final product assets
- reviewed docs
- design prototypes that are intentionally committed

The design source is allowed to be messy while exploring. The repo should be tidy and predictable.

## 4. Recommended Design-Source Structure

Recommended local-only structure:

```text
01_design-source/
  legacy/
  openart/
  runway/
  figma-exports/
  videos/
  references/
  final-assets/
```

Suggested usage:

```text
legacy/
  Older workspaces and recovered project folders.

openart/
  OpenArt drafts, prompts, generated images, and generated videos.

runway/
  Runway drafts, motion experiments, and generated video outputs.

figma-exports/
  Figma PNG/SVG exports and temporary screen exports.

videos/
  Video edits, loops, conversions, and test renders.

references/
  Inspiration, competitor screenshots, visual direction notes, and research.

final-assets/
  Assets that have been selected and are ready to be copied into the repo.
```

## 5. Recommended Repo Prototype Structure

Design prototypes that should be reviewed in GitHub can live under:

```text
design/prototypes/
```

For the Chrome extension UI prototype:

```text
design/prototypes/chrome-extension/
  manifest.json
  background.js
  sidepanel.html
  src/
  public/assets/
```

This folder is for review and design validation. It is not the production extension app.

The production extension app lives under:

```text
apps/extension/
```

Do not overwrite `apps/extension/` with prototype files unless the product direction has been confirmed.

## 6. Stable Names Inside the Repo

Design-source files may include version names:

```text
crab-say-hi-loop-final-v03.mp4
login-character-v05.svg
opening-loop-selected-v02.mp4
```

Once copied into the repo, use stable product names:

```text
crab-say-hi-loop.mp4
login-character.svg
opening-01-loop.mp4
```

Reason: frontend imports should not need to change every time a design version changes.

## 7. Version Naming in Design Source Only

Version numbers are allowed in design source:

```text
v01
v02
v03
final-v01
final-v02
```

Examples:

```text
01_design-source/openart/crab-say-hi/drafts/crab-say-hi-loop-v01.mp4
01_design-source/openart/crab-say-hi/drafts/crab-say-hi-loop-v02.mp4
01_design-source/openart/crab-say-hi/final/crab-say-hi-loop-final-v03.mp4
```

Avoid version names in the production repo unless the product intentionally needs multiple versions at runtime.

## 8. Do Not Commit Drafts

Do not commit:

- OpenArt drafts
- Runway drafts
- Figma export experiments
- unused screenshots
- temporary mockups
- large source design files
- random generated filenames
- `.DS_Store`

Folders that should usually stay local or ignored:

```text
openart-drafts/
runway-drafts/
figma-exports-draft/
_design-drafts/
archive/
```

Only commit final assets or intentionally reviewed prototypes.

## 9. Final Extension Asset Structure

When assets become part of the production extension, prefer:

```text
apps/extension/public/assets/
  brand/
    tachigo-logo.svg
    tachigo-wordmark.svg

  opening/
    opening-01-loop.mp4
    opening-02-loop.mp4

  backgrounds/
    login-bg.png
    mining-bg.png
    character-select-bg.png

  characters/
    crab/
      crab-idle.png
      crab-say-hi-loop.mp4
      crab-mining-loop.mp4
    tama/
      tama-idle.png
    propo/
      propo-idle.png
    hidden/
      hidden-character.png

  icons/
    arrow-left.svg
    arrow-right.svg
    close.svg
    settings.svg
    notice.svg
    twitch.svg

  ui/
    panel-glow.png
    button-highlight.png
```

## 10. Git LFS Guidance

Large videos and large raster images should use Git LFS.

Recommended LFS patterns:

```text
apps/extension/public/assets/**/*.mp4
apps/extension/public/assets/**/*.mov
apps/extension/public/assets/**/*.webm
apps/extension/public/assets/**/*.gif
apps/extension/public/assets/**/*.png
apps/extension/public/assets/**/*.jpg
apps/extension/public/assets/**/*.jpeg
design/prototypes/**/*.mp4
```

SVG files usually stay in normal Git because they are text-based and easier to diff.

Small UI icons should usually be SVG and should not use Git LFS.

## 11. Component and Screen Naming

For React component files, use kebab-case file names:

```text
login-screen.tsx
character-select-screen.tsx
crab-say-hi-screen.tsx
claim-panel.tsx
balance-display.tsx
icon-button.tsx
```

Inside the file, React component names should use PascalCase:

```text
LoginScreen
CharacterSelectScreen
CrabSayHiScreen
ClaimPanel
BalanceDisplay
IconButton
```

## 12. Codex Working Rule

When Codex works on extension production tasks, it should primarily work inside:

```text
apps/extension/
```

For production asset updates, it should work inside:

```text
apps/extension/public/assets/
```

For design prototype review, it may work inside:

```text
design/prototypes/
```

Codex should not depend on local-only design-source folders at runtime.
