package epistemic

import "encoding/json"

type eventWire Event

var eventFields = map[string]bool{"spec_version": true, "id": true, "type": true, "source": true, "subject": true, "time": true, "context": true, "ordering": true, "idempotency_key": true, "data": true, "extensions": true}

func (event *Event) UnmarshalJSON(data []byte) error {
	var wire eventWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	wire.Unknown = map[string]json.RawMessage{}
	for key, value := range fields {
		if !eventFields[key] {
			wire.Unknown[key] = value
		}
	}
	*event = Event(wire)
	return nil
}

func (event Event) MarshalJSON() ([]byte, error) {
	wire := eventWire(event)
	wire.Unknown = nil
	data, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	if len(event.Unknown) == 0 {
		return data, nil
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for key, value := range event.Unknown {
		if !eventFields[key] {
			fields[key] = value
		}
	}
	return json.Marshal(fields)
}
