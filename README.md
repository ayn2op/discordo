# discordo &middot; [![ci](https://github.com/ayntgl/discordo/actions/workflows/ci.yml/badge.svg)](https://github.com/ayntgl/discordo/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/ayntgl/discordo)](https://goreportcard.com/report/github.com/ayntgl/discordo) [![license](https://img.shields.io/github/license/ayntgl/discordo?logo=github)](https://github.com/ayntgl/discordo/blob/master/LICENSE)

Discordo is a lightweight, secure, and feature-rich Discord terminal client. Heavily work-in-progress, expect breaking changes.

![Preview](.github/preview.png)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
  - [Prebuilt binaries](#prebuilt-binaries)
  - [Package managers](#package-managers)
  - [Building from source](#building-from-source)
- [Usage](#usage)
  - [Configuration](#configuration)

## Features

- Lightweight
- Secure
- Configurable
- Cross-platform
- Minimalistic
- Feature-rich
  - Mouse & clipboard support
  - 2-Factor authentication
  - Desktop notifications
  - Partial [Discord-flavored markdown](https://support.discord.com/hc/en-us/articles/210298617-Markdown-Text-101-Chat-Formatting-Bold-Italic-Underline-)

## Installation

## Prebuilt binaries

You can download and install a [prebuilt binary here](https://nightly.link/ayntgl/discordo/workflows/ci/main) for Windows, macOS, or Linux.

### Package managers

- [Arch Linux](https://aur.archlinux.org/packages/discordo-git/): `yay -S discordo-git`

### Building from source

```bash
git clone https://github.com/ayntgl/discordo
cd discordo
make build

# optional
sudo mv ./discordo /usr/local/bin
```

### Linux clipboard support

- `xclip` or `xsel` for X11.
  - Ubuntu: `apt install xclip`
  - Arch Linux: `pacman -S xclip`
  - Fedora: `dnf install xclip`
- `wl-clipboard` for Wayland.
  - Ubuntu: `apt install wl-clipboard`
  - Arch Linux: `pacman -S wl-clipboard`
  - Fedora: `dnf install wl-clipboard`

## Usage

- Run the `discordo` executable with no arguments.

- Log in using the account email and password (first-time login) and click on the "Login" button to continue.

  - If you are using a bot account to login, set the `DISCORDO_TOKEN` environment variable to the token of the bot and prefix it with `Bot ` (eg: `Bot OTI2MDU5NTQxNDE2Nzc5ODA2.Yc2KKA.2iZ-5JxgxG-9Ub8GHzBSn-NJjNg`).

  - By default, Discordo utilizes OS-specific keyring to store the authentication token. However, if you prefer not to use a keyring (not recommended), you may set the `DISCORDO_TOKEN` environment variable and Discordo will prioritize the usage of the provided token to login instead of keyring.

### Configuration

A default configuration file is created on first start-up at `$HOME/.config/discordo.toml` on Unix, `$HOME/Library/Application Support/discordo.toml` on Darwin, and `%AppData%/discordo.toml` on Windows.

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
