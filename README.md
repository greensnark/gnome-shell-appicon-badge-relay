# Gnome Shell App Icon Badging Relay

A relay HTTP service that listens on localhost and emits dbus notifications for
received messages.

The intent is to allow local services to request that Gnome Shell badges an app
window by passing in a window instance ID and badge parameters.