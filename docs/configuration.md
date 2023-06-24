# Configuration

The configuration file allows you to customize behavior, keybindings, and theme of the application. It is created on first start-up at the following location:

| Operating System | Configuration File Location                             |
| ---------------- | ------------------------------------------------------- |
| Unix             | `$HOME/.config/discordo/config.yml`                     |
| Darwin           | `$HOME/Library/Application Support/discordo/config.yml` |
| Windows          | `%AppData%/discordo/config.yml`                         |

Optionally, you can specify a different location for configuration file with the `config` command-line flag to the executable (eg: `--config foo/bar/conf.yml`).

## Keybindings

Keybindings are configurable in the [configuration file](#configuration).

| Action                | Keybinding |
| --------------------- | ---------- |
| **Guilds Tree**       |            |
| Focus                 | Alt + g    |
| Toggle                | Alt + b    |
|                       |            |
| **Messages Text**     |            |
| Focus                 | Alt + m    |
| Show image            | i          |
| Copy message content  | c          |
| Reply without mention | r          |
| Reply with mention    | R          |
| Select reply          | s          |
| Select previous       | Up arrow   |
| Select next           | Down arrow |
| Select first          | Home       |
| Select last           | End        |
|                       |            |
| **Message Input**     |            |
| Focus                 | Alt + i    |
| Send message          | Enter      |
| Paste from clipboard  | Ctrl + v   |
| Launch editor         | Ctrl + e   |
