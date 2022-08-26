local string = require "string"

-- Whether the mouse is usable or not.
mouse = true

-- The maximum number of messages to fetch and display on the messages panel.
-- Its value must not be lesser than 1 and greater than 100.
messagesLimit = 50

-- Whether to display the timestamp of the message beside the displayed message or not.
timestamps = false

-- The timezone of the timestamps.

-- If its value is "" or "UTC", UTC is used. Learn more: https://pkg.go.dev/time#UTC
-- If its value is "Local", local timezone is used. Learn more: https://pkg.go.dev/time#Local

-- Otherwise, its value is taken to be a location name corresponding to a file in the IANA Time Zone database, such as "America/New_York".

-- It looks for the IANA Time Zone database in the following locations in order:
-- - the directory or uncompressed zip file named by the ZONEINFO environment variable
-- - on a Unix system, the system standard installation location
-- - $GOROOT/lib/time/zoneinfo.zi
-- - the time/tzdata package, if it was imported
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
    userAgent = string.format("Mozilla/5.0 (X11; %s x86_64) AppleWebKit/537.36 (KHTML, like Gecko) %s/%s Safari/537.36", oss, browser, browserVersion),
    browser = browser,
    browserVersion = browserVersion,
    os = oss
}

keys = {
    application = {
        focusGuildsTree = "Rune[g]",
        focusChannelsTree = "Rune[c]",
        focusMessagesPanel = "Rune[m]",
        focusMessageInput = "Rune[i]"
    },
    messagesPanel = {
        openMessageActionsList = "Rune[a]",

        selectPreviousMessage = "Up",
        selectNextMessage = "Down",
        selectFirstMessage = "Home",
        selectLastMessage = "End"
    },
    messageInput = {openExternalEditor = "Ctrl+E"}
}

theme = {background = "black", border = "white", title = "white"}
