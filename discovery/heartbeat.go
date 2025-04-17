package discovery

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
)

const DefaultHeartbeatFrequency = 5 * time.Minute

var ErrNoHealthcheckDefined = errors.New("no healthcheck defined")

// HeartbeatSender sends a heartbeat to the management API, this is called at
// `DefaultHeartbeatFrequency` by default when the engine is running, or
// `StartSendingHeartbeats` has been called manually. Users can also call this
// method to immediately send a heartbeat if required
func (e *Engine) SendHeartbeat(ctx context.Context) error {
	if e.EngineConfig.HeartbeatOptions == nil || e.EngineConfig.HeartbeatOptions.HealthCheck == nil {
		return ErrNoHealthcheckDefined
	}

	healthCheckError := e.EngineConfig.HeartbeatOptions.HealthCheck(ctx)

	var heartbeatError *string

	if healthCheckError != nil {
		heartbeatError = new(string)
		*heartbeatError = healthCheckError.Error()
	}

	var engineUUID []byte

	if e.EngineConfig.SourceUUID != uuid.Nil {
		engineUUID = e.EngineConfig.SourceUUID[:]
	}

	availableScopes, adapterMetadata := e.GetAvailableScopesAndMetadata()

	// Calculate the duration for the next heartbeat, based on the current
	// frequency x2.5 to give us some leeway
	nextHeartbeat := time.Duration(float64(e.EngineConfig.HeartbeatOptions.Frequency) * 2.5)

	_, err := e.EngineConfig.HeartbeatOptions.ManagementClient.SubmitSourceHeartbeat(ctx, &connect.Request[sdp.SubmitSourceHeartbeatRequest]{
		Msg: &sdp.SubmitSourceHeartbeatRequest{
			UUID:             engineUUID,
			Version:          e.EngineConfig.Version,
			Name:             e.EngineConfig.SourceName,
			Type:             e.EngineConfig.EngineType,
			AvailableScopes:  availableScopes,
			AdapterMetadata:  adapterMetadata,
			Managed:          e.EngineConfig.OvermindManagedSource,
			Error:            heartbeatError,
			NextHeartbeatMax: durationpb.New(nextHeartbeat),
		},
	})

	return err
}

// Starts sending heartbeats at the specified frequency. These will be sent in
// the background and this function will return immediately. Heartbeats are
// automatically started when the engine started, but if an adapter has startup
// steps that take a long time, or are liable to fail, the user may want to
// start the heartbeats first so that users can see that the adapter has failed
// to start.
//
// If this is called multiple times, nothing will happen. Heartbeats will be
// stopped when the engine is stopped, or when the provided context is canceled.
//
// This will send one heartbeat initially when the method is called, and will
// then run in a background goroutine that sends heartbeats at the specified
// frequency, and will stop when the provided context is canceled.
func (e *Engine) StartSendingHeartbeats(ctx context.Context) {
	if e.EngineConfig.HeartbeatOptions == nil || e.EngineConfig.HeartbeatOptions.Frequency == 0 || e.heartbeatCancel != nil {
		return
	}

	var heartbeatContext context.Context
	heartbeatContext, e.heartbeatCancel = context.WithCancel(ctx)

	// Send one heartbeat at the beginning
	err := e.SendHeartbeat(heartbeatContext)
	if err != nil {
		log.WithError(err).Error("Failed to send heartbeat")
	}

	go func() {
		ticker := time.NewTicker(e.EngineConfig.HeartbeatOptions.Frequency)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatContext.Done():
				return
			case <-ticker.C:
				err := e.SendHeartbeat(heartbeatContext)
				if err != nil {
					log.WithError(err).Error("Failed to send heartbeat")
				}
			}
		}
	}()
}
