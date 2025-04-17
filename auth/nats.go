package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"

	"github.com/nats-io/nats.go"
)

// Defaults
const MaxReconnectsDefault = -1
const ReconnectWaitDefault = 1 * time.Second
const ReconnectJitterDefault = 5 * time.Second
const ConnectionTimeoutDefault = 10 * time.Second

type MaxRetriesError struct{}

func (m MaxRetriesError) Error() string {
	return "maximum retries reached"
}

func fieldsFromConn(c *nats.Conn) log.Fields {
	fields := log.Fields{}

	if c != nil {
		fields["ovm.nats.address"] = c.ConnectedAddr()
		fields["ovm.nats.reconnects"] = c.Reconnects
		fields["ovm.nats.serverId"] = c.ConnectedServerId()
		fields["ovm.nats.url"] = c.ConnectedUrl()

		if c.LastError() != nil {
			fields["ovm.nats.lastError"] = c.LastError()
		}
	}

	return fields
}

var DisconnectErrHandlerDefault = func(c *nats.Conn, err error) {
	fields := fieldsFromConn(c)

	if err != nil {
		log.WithError(err).WithFields(fields).Error("NATS disconnected")
	} else {
		log.WithFields(fields).Debug("NATS disconnected")
	}
}

var ConnectHandlerDefault = func(c *nats.Conn) {
	fields := fieldsFromConn(c)

	log.WithFields(fields).Debug("NATS connected")
}
var ReconnectHandlerDefault = func(c *nats.Conn) {
	fields := fieldsFromConn(c)

	log.WithFields(fields).Debug("NATS reconnected")
}
var ClosedHandlerDefault = func(c *nats.Conn) {
	fields := fieldsFromConn(c)

	log.WithFields(fields).Debug("NATS connection closed")
}
var LameDuckModeHandlerDefault = func(c *nats.Conn) {
	fields := fieldsFromConn(c)

	log.WithFields(fields).Debug("NATS server has entered lame duck mode")
}
var ErrorHandlerDefault = func(c *nats.Conn, s *nats.Subscription, err error) {
	fields := fieldsFromConn(c)

	if s != nil {
		fields["ovm.nats.subject"] = s.Subject
		fields["ovm.nats.queue"] = s.Queue
	}

	log.WithFields(fields).WithError(err).Error("NATS error")
}

type NATSOptions struct {
	Servers              []string            // List of server to connect to
	ConnectionName       string              // The client name
	MaxReconnects        int                 // The maximum number of reconnect attempts
	ConnectionTimeout    time.Duration       // The timeout for Dial on a connection
	ReconnectWait        time.Duration       // Wait time between reconnect attempts
	ReconnectJitter      time.Duration       // The upper bound of a random delay added ReconnectWait
	TokenClient          TokenClient         // The client to use to get NATS tokens
	ConnectHandler       nats.ConnHandler    // Runs when NATS is connected
	DisconnectErrHandler nats.ConnErrHandler // Runs when NATS is disconnected
	ReconnectHandler     nats.ConnHandler    // Runs when NATS has successfully reconnected
	ClosedHandler        nats.ConnHandler    // Runs when NATS will no longer be connected
	ErrorHandler         nats.ErrHandler     // Runs when there is a NATS error
	LameDuckModeHandler  nats.ConnHandler    // Runs when the connection enters "lame duck mode"
	AdditionalOptions    []nats.Option       // Addition options to pass to the connection
	NumRetries           int                 // How many times to retry connecting initially, use -1 to retry indefinitely
	RetryDelay           time.Duration       // Delay between connection attempts
}

// Creates a copy of the NATS options, **excluding** the token client as these
// should not be re-used
func (o NATSOptions) Copy() NATSOptions {
	return NATSOptions{
		Servers:              o.Servers,
		ConnectionName:       o.ConnectionName,
		MaxReconnects:        o.MaxReconnects,
		ConnectionTimeout:    o.ConnectionTimeout,
		ReconnectWait:        o.ReconnectWait,
		ReconnectJitter:      o.ReconnectJitter,
		ConnectHandler:       o.ConnectHandler,
		DisconnectErrHandler: o.DisconnectErrHandler,
		ReconnectHandler:     o.ReconnectHandler,
		ClosedHandler:        o.ClosedHandler,
		LameDuckModeHandler:  o.LameDuckModeHandler,
		ErrorHandler:         o.ErrorHandler,
		AdditionalOptions:    o.AdditionalOptions,
		NumRetries:           o.NumRetries,
		RetryDelay:           o.RetryDelay,
	}
}

// ToNatsOptions Converts the struct to connection string and a set of NATS
// options
func (o NATSOptions) ToNatsOptions() (string, []nats.Option) {
	serverString := strings.Join(o.Servers, ",")
	options := []nats.Option{}

	if o.ConnectionName != "" {
		options = append(options, nats.Name(o.ConnectionName))
	}

	if o.MaxReconnects != 0 {
		options = append(options, nats.MaxReconnects(o.MaxReconnects))
	} else {
		options = append(options, nats.MaxReconnects(MaxReconnectsDefault))
	}

	if o.ConnectionTimeout != 0 {
		options = append(options, nats.Timeout(o.ConnectionTimeout))
	} else {
		options = append(options, nats.Timeout(ConnectionTimeoutDefault))
	}

	if o.ReconnectWait != 0 {
		options = append(options, nats.ReconnectWait(o.ReconnectWait))
	} else {
		options = append(options, nats.ReconnectWait(ReconnectWaitDefault))
	}

	if o.ReconnectJitter != 0 {
		options = append(options, nats.ReconnectJitter(o.ReconnectJitter, o.ReconnectJitter))
	} else {
		options = append(options, nats.ReconnectJitter(ReconnectJitterDefault, ReconnectJitterDefault))
	}

	if o.TokenClient != nil {
		options = append(options, nats.UserJWT(func() (string, error) {
			return o.TokenClient.GetJWT()
		}, o.TokenClient.Sign))
	}

	if o.ConnectHandler != nil {
		options = append(options, nats.ConnectHandler(o.ConnectHandler))
	} else {
		options = append(options, nats.ConnectHandler(ConnectHandlerDefault))
	}

	if o.DisconnectErrHandler != nil {
		options = append(options, nats.DisconnectErrHandler(o.DisconnectErrHandler))
	} else {
		options = append(options, nats.DisconnectErrHandler(DisconnectErrHandlerDefault))
	}

	if o.ReconnectHandler != nil {
		options = append(options, nats.ReconnectHandler(o.ReconnectHandler))
	} else {
		options = append(options, nats.ReconnectHandler(ReconnectHandlerDefault))
	}

	if o.ClosedHandler != nil {
		options = append(options, nats.ClosedHandler(o.ClosedHandler))
	} else {
		options = append(options, nats.ClosedHandler(ClosedHandlerDefault))
	}

	if o.LameDuckModeHandler != nil {
		options = append(options, nats.LameDuckModeHandler(o.LameDuckModeHandler))
	} else {
		options = append(options, nats.LameDuckModeHandler(LameDuckModeHandlerDefault))
	}

	if o.ErrorHandler != nil {
		options = append(options, nats.ErrorHandler(o.ErrorHandler))
	} else {
		options = append(options, nats.ErrorHandler(ErrorHandlerDefault))
	}

	options = append(options, o.AdditionalOptions...)

	return serverString, options
}

// ConnectAs Connects to NATS using the supplied options, including retrying if
// unavailable
func (o NATSOptions) Connect() (sdp.EncodedConnection, error) {
	servers, opts := o.ToNatsOptions()

	var triesLeft int

	if o.NumRetries >= 0 {
		triesLeft = o.NumRetries + 1
	} else {
		triesLeft = -1
	}

	var nc *nats.Conn
	var err error

	for triesLeft != 0 {
		triesLeft--
		lf := log.Fields{
			"servers":   servers,
			"triesLeft": triesLeft,
		}
		log.WithFields(lf).Info("NATS connecting")

		nc, err = nats.Connect(
			servers,
			opts...,
		)

		if err != nil && triesLeft != 0 {
			log.WithError(err).WithFields(lf).Error("Error connecting to NATS")
			time.Sleep(o.RetryDelay)
			continue
		}

		log.WithFields(lf).Info("NATS connected")
		break
	}

	if err != nil {
		err = errors.Join(err, MaxRetriesError{})
		return &sdp.EncodedConnectionImpl{}, err
	}

	return &sdp.EncodedConnectionImpl{Conn: nc}, nil
}
