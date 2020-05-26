package freeswitchesl

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gitlab.percipia.com/libs/go/freeswitchesl/command"
	"gitlab.percipia.com/libs/go/freeswitchesl/command/call"
)

func (c *Conn) EnableEvents(ctx context.Context) error {
	var err error
	if c.outbound {
		_, err = c.SendCommand(ctx, command.MyEvents{
			Format: "plain",
		})
	} else {
		_, err = c.SendCommand(ctx, command.Event{
			Format: "plain",
			Listen: []string{"all"},
		})
	}
	return err
}

func (c *Conn) OriginateCall(ctx context.Context, aLeg, bLeg string, vars map[string]string) (string, error) {
	if vars == nil {
		vars = make(map[string]string)
	}
	vars["origination_uuid"] = uuid.New().String()

	_, err := c.SendCommand(ctx, command.API{
		Command:    "originate",
		Arguments:  fmt.Sprintf("%s%s %s", buildVars(vars), aLeg, bLeg),
		Background: true,
	})
	if err != nil {
		return vars["origination_uuid"], err
	}
	return vars["origination_uuid"], nil
}

func (c *Conn) HangupCall(ctx context.Context, uuid, cause string) error {
	_, err := c.SendCommand(ctx, call.Hangup{
		UUID:  uuid,
		Cause: cause,
		Sync:  false,
	})
	return err
}

func (c *Conn) AnswerCall(ctx context.Context, uuid string) error {
	_, err := c.SendCommand(ctx, &call.Execute{
		UUID:    uuid,
		AppName: "answer",
		Sync:    true,
	})
	return err
}

func (c *Conn) Playback(ctx context.Context, uuid, audioCommand string, times int, wait bool) error {
	response, err := c.SendCommand(ctx, &call.Execute{
		UUID:    uuid,
		AppName: "playback",
		AppArgs: audioCommand,
		Sync:    wait,
	})
	if err != nil {
		return err
	}
	if !response.IsOk() {
		return errors.New("playback response is not okay")
	}
	return nil
}

// WaitForDTMF, waits for a DTMF event. Requires events to be enabled!
func (c *Conn) WaitForDTMF(ctx context.Context, uuid string) (byte, error) {
	done := make(chan byte, 1)
	listenerID := c.RegisterEventListener(uuid, func(event *Event) {
		if event.Headers.Get("Event-Name") == "DTMF" {
			dtmf := event.Headers.Get("DTMF-Digit")
			if len(dtmf) > 0 {
				done <- dtmf[0]
			}
			done <- 0
		}
	})
	defer c.RemoveEventListener(uuid, listenerID)

	select {
	case digit := <-done:
		if digit != 0 {
			return digit, nil
		}
		return digit, errors.New("invalid DTMF digit received")
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
