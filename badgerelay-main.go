package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	bindEnv(notificationServerCommand()).Execute()
}

func bindEnv(c *cobra.Command) *cobra.Command {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("GSBADGE")
	viper.AutomaticEnv()
	viper.BindPFlags(c.Flags())
	return c
}

func notificationServerCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   os.Args[0],
		Short: "Listens for HTTP requests to badge apps in the Gnome Shell dock and relays the requests to dbus",
		Run:   listenForNotifications,
	}

	flags := c.Flags()
	flags.Int("port", 18989, "port to start http->dbus relay service on")
	flags.String("host", "localhost", "host interface to start http->dbus relay service on")
	flags.String("dest", "org.gnome.Shell", "dbus destination")
	flags.String("field-path", "/org/shalott/dbus/DockIcon", "dbus path")
	flags.String("field-interface", "org.shalott.dbus.DockIcon", "dbus interface name (excluding method name)")
	flags.String("field-member", "SetAppNotifications", "dbus method name (excluding interface name)")

	return c
}

func listenForNotifications(c *cobra.Command, args []string) {
	relay, err := NewDBusHTTPRelay(
		RelayLog(
			log.With(log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr)),
				"ts", log.DefaultTimestamp)),
		RelayHost(viper.GetString("host")),
		RelayPort(viper.GetInt("port")),
		RelayDest(viper.GetString("dest")),
		RelayPath(viper.GetString("field-path")),
		RelayInterface(viper.GetString("field-interface")),
		RelayMethod(viper.GetString("field-member")))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating http -> DBus relay:", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Starting http -> DBus relay on", relay.ListenAddr())
	http.ListenAndServe(relay.ListenAddr(), relay.ServeMux())
}
