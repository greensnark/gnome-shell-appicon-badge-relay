package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/godbus/dbus"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

type DBusHTTPRelay struct {
	log log.Logger

	host string
	port int

	dbus            *dbus.Conn
	dbusDestination string
	dbusPath        string
	dbusInterface   string
	dbusMethod      string
}

func (d *DBusHTTPRelay) configError() (err error) {
	if d.dbus != nil {
		return nil
	}

	d.dbus, err = dbus.SessionBus()
	return errors.Wrap(err, "could not fetch dbus SessionBus")
}

func (d *DBusHTTPRelay) ListenAddr() string {
	return fmt.Sprintf("%s:%d", d.host, d.port)
}

type dbusNotificationRequest struct {
	Label string `json:"label"`
	Color string `json:"color"`
}

func (d *DBusHTTPRelay) ServeMux() http.Handler {
	mux := httprouter.New()
	mux.POST("/:windowID", d.setWindowNotifications)
	return mux
}

func (d *DBusHTTPRelay) raiseDBusSignal(windowID string, notification dbusNotificationRequest) error {
	values := []interface{}{windowID, notification.Label, notification.Color}
	msg := &dbus.Message{
		Type: dbus.TypeSignal,
		Headers: map[dbus.HeaderField]dbus.Variant{
			dbus.FieldInterface:   dbus.MakeVariant(d.dbusInterface),
			dbus.FieldMember:      dbus.MakeVariant(d.dbusMethod),
			dbus.FieldPath:        dbus.MakeVariant(dbus.ObjectPath(d.dbusPath)),
			dbus.FieldSignature:   dbus.MakeVariant(dbus.SignatureOf(values...)),
			dbus.FieldDestination: dbus.MakeVariant(d.dbusDestination),
		},
		Body: values,
	}

	call := d.dbus.Send(msg, nil)
	return call.Err
}

func resultText(err error) string {
	if err == nil {
		return "OK"
	}

	return "err:" + err.Error()
}

func (d *DBusHTTPRelay) setWindowNotifications(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	badRequest := func(message string) {
		http.Error(w, message, http.StatusBadRequest)
	}

	windowID := strings.TrimSpace(params.ByName("windowID"))
	if windowID == "" {
		badRequest("missing window ID")
		return
	}

	var notificationRequest dbusNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&notificationRequest); err != nil {
		badRequest("malformed body")
		return
	}

	err := d.raiseDBusSignal(windowID, notificationRequest)
	d.log.Log("windowID", windowID, "label", notificationRequest.Label, "color", notificationRequest.Color,
		"result", resultText(err))

	if err != nil {
		http.Error(w, fmt.Sprint("unable to raise signal:", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type RelaySetting func(*DBusHTTPRelay)

func NewDBusHTTPRelay(settings ...RelaySetting) (*DBusHTTPRelay, error) {
	relay := &DBusHTTPRelay{
		log: log.NewNopLogger(),
	}
	for _, setting := range settings {
		setting(relay)
	}
	if err := relay.configError(); err != nil {
		return nil, err
	}

	return relay, nil
}

func RelayLog(log log.Logger) RelaySetting {
	return func(d *DBusHTTPRelay) { d.log = log }
}

func RelayDestinationBus(bus *dbus.Conn) RelaySetting {
	return func(d *DBusHTTPRelay) { d.dbus = bus }
}

func RelayHost(host string) RelaySetting {
	return func(d *DBusHTTPRelay) { d.host = host }
}

func RelayPort(port int) RelaySetting {
	return func(d *DBusHTTPRelay) { d.port = port }
}

func RelayDest(dest string) RelaySetting {
	return func(d *DBusHTTPRelay) { d.dbusDestination = dest }
}

func RelayPath(path string) RelaySetting {
	return func(d *DBusHTTPRelay) { d.dbusPath = path }
}

func RelayInterface(interfaceName string) RelaySetting {
	return func(d *DBusHTTPRelay) { d.dbusInterface = interfaceName }
}

func RelayMethod(method string) RelaySetting {
	return func(d *DBusHTTPRelay) { d.dbusMethod = method }
}
