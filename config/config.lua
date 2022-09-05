local string = require "string"

-- Whether the mouse is usable or not.
mouse = true

-- The maximum number of messages to fetch and display on the messages panel.
-- Its value must not be lesser than 1 and greater than 100.
messagesLimit = 50

-- Whether to display the timestamp of the message beside the displayed message or not.
timestamps = false

-- The timezone of the timestamps.
-- Learn more: https://pkg.go.dev/time#LoadLocation
timezone = "Local"

-- A textual representation of the time value formatted according to the layout defined by its value.
-- Learn more: https://pkg.go.dev/time#Layout
timeFormat = "3:04PM"

browser = "Chrome"
browserVersion = "104.0.5112.102"
oss = "Linux"

-- Identify properties are connection properties that are dispatched in the IDENTIFY gateway event to trigger the initial handshake with the gateway.
-- Learn more: https://discord.com/developers/docs/topics/gateway#identify
identifyProperties = {
    userAgent = string.format(
        "Mozilla/5.0 (X11; %s x86_64) AppleWebKit/537.36 (KHTML, like Gecko) %s/%s Safari/537.36",
        oss,
        browser,
        browserVersion
    ),
    browser = browser,
    browserVersion = browserVersion,
    os = oss
}

-- Keybindings
keys = {
    -- application = {
    --     key(
    --         "Ctrl+R",
    --         "Refresh the screen.",
    --         function(core, event)
    --             core.Application:Sync()
    --             return nil
    --         end
    --     )
    -- },
    messagesPanel = {
        key(
            "Rune[a]",
            "Open the message actions list widget.",
            function(core, event)
                return openMessageActionsList()
            end
        ),
        key(
            "Up",
            "Select the previous message.",
            function(core, event)
                return selectPreviousMessage()
            end
        ),
        key(
            "Down",
            "Select the next message.",
            function(core, event)
                return selectNextMessage()
            end
        ),
        key(
            "Home",
            "Select the first message.",
            function(core, event)
                return selectFirstMessage()
            end
        ),
        key(
            "End",
            "Select the last message.",
            function(core, event)
                return selectLastMessage()
            end
        )
    },
    messageInput = {
        key(
            "Ctrl+E",
            "Open the external editor.",
            function()
                return openExternalEditor()
            end
        ),
        key(
            "Ctrl+V",
            "Paste the clipboard content.",
            function()
                return pasteClipboardContent()
            end
        )
    }
}

-- Theme
theme = {background = "default", border = "white", title = "white"}
