# Discordo &middot; [![ci](https://github.com/ayn2op/discordo/actions/workflows/ci.yml/badge.svg)](https://github.com/ayn2op/discordo/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/ayn2op/discordo)](https://goreportcard.com/report/github.com/ayn2op/discordo) [![license](https://img.shields.io/github/license/ayn2op/discordo?logo=github)](https://github.com/ayn2op/discordo/blob/master/LICENSE)

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
  - [Controls](#controls)
- [Disclaimer](#disclaimer)

## Features

- Lightweight
- Secure
- Configurable
- Cross-platform
- Minimalistic
- Feature-rich
  - Mouse & clipboard support
  - 2-Factor authentication
  - Partial [Discord-flavored markdown](https://support.discord.com/hc/en-us/articles/210298617-Markdown-Text-101-Chat-Formatting-Bold-Italic-Underline-)

## Installation

### Prebuilt binaries

You can download and install a [prebuilt binary here](https://nightly.link/ayn2op/discordo/workflows/ci/main) for Windows, macOS, or Linux.

### Package managers

- [Arch Linux](https://aur.archlinux.org/packages/discordo-git/): `yay -S discordo-git` (thanks to [Alyxia Sother](https://github.com/lexisother) for maintaining the AUR package).
- [FreeBSD](https://www.freshports.org/net-im/discordo): `pkg install discordo` or via the ports system `make -C /usr/ports/net-im/discordo install clean`.

### Building from source

```bash
git clone https://github.com/ayn2op/discordo
cd discordo
go build .

# optional
sudo mv ./discordo /usr/local/bin
```

### Linux clipboard support

- `xclip` or `xsel` for X11 (`apt install xclip`)
- `wl-clipboard` for Wayland (`apt install wl-clipboard`)

## Usage

1. Run the `discordo` executable with no arguments.

- If you are logging in using an authentication token, provide the `token` command-line flag to the executable (eg: `--token "OTI2MDU5NTQxNDE2Nzc5ODA2.Yc2KKA.2iZ-5JxgxG-9Ub8GHzBSn-NJjNg"`). The token is stored securely in the default OS-specific keyring.

2. Enter your email and password and click on the "Login" button to continue.

- Most of the Discord third-party clients store the token in a configuration file unencrypted. Discordo securely stores the token in the default OS-specific keyring.

### Configuration

A default configuration file is created on first start-up at `$HOME/.config/discordo/config.yml` on Unix, `$HOME/Library/Application Support/discordo/config.yml` on Darwin, and `%AppData%/discordo/config.yml` on Windows.

Similarly, a log file is created on first start-up at `$HOME/.cache/discordo/logs.txt` on Unix, `$HOME/Library/Caches/discordo/logs.txt` on Darwin, and `%LocalAppData%/discordo/logs.txt` on Windows.

### Controls
##### Focusing
* `Alt-g` - Focus into server list
* `Alt-m` - Focus into channel messages list
* `Alt-i` - Focus into message box
##### Navigating & Selecting
* `J/K` + (Channel Messages Focused) - Scroll down and up channel messages
* `J/K` + (Server List Focused) - Navigate Server List
* `Up arrow` + (Channel Messages Focused) - Select Prev Message
* `Down arrow` + (Channel Messages Focused) - Select Next Message
* `Up arrow` + (Channel Messages Focused) - Select Prev Message
* `Home` + (Channel Messages Focused) - Select First Message
* `End` + (Channel Messages Focused) - Select Last Message
##### Actions
* `i` + (Message Selected) - View image (In terminal resolution)
* `Esc` + (Image View Selected) - Escape image view
* `r` + (Message Selected) - Reply (No ping)
* `R` + (Message Selected) - Reply (Ping)
* `s` + (Message Selected) - Jump to the message in which that message replied to
* `c` + (Message Selected) - Copy Message
* `Ctrl-Shift-v` + (Message Box Selected) - Paste Into Message Box
* `Ctrl-Shift-e` + (Message Box Selected) - Open external editor (Needs [Configuration](#configuration))

## Disclaimer

Automated user accounts or "self-bots" are against Discord's Terms of Service. I am not responsible for any loss caused by using "self-bots" or Discordo.
