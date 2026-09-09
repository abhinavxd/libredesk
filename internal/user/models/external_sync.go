package models

import "encoding/json"

// Identity fields the widget integration supplies through the JWT and the desk remembers in users.external_sync.
const (
	ExternalSyncFirstName        = "first_name"
	ExternalSyncLastName         = "last_name"
	ExternalSyncEmail            = "email"
	ExternalSyncPhoneNumber      = "phone_number"
	ExternalSyncPhoneCountryCode = "phone_number_country_code"
)

// ExternalSyncFields are the contact fields an integration may keep in sync through the JWT.
var ExternalSyncFields = []string{
	ExternalSyncFirstName,
	ExternalSyncLastName,
	ExternalSyncEmail,
	ExternalSyncPhoneNumber,
	ExternalSyncPhoneCountryCode,
}

// UnmarshalExternalSync reads the identity an integration last supplied for a contact.
// A missing or unreadable record simply means nothing was supplied yet.
func UnmarshalExternalSync(raw json.RawMessage) map[string]string {
	synced := map[string]string{}
	if len(raw) == 0 {
		return synced
	}
	if err := json.Unmarshal(raw, &synced); err != nil {
		return map[string]string{}
	}
	// An absent field and one supplied empty mean the same thing here; keeping only one of the
	// two forms lets the caller compare the old and the new record for equality.
	for field, value := range synced {
		if value == "" {
			delete(synced, field)
		}
	}
	return synced
}

// ResolveExternalSync decides which integration-supplied identity fields may overwrite a contact.
//
// current is what the contact holds now, synced is what the integration last supplied, and incoming
// is what it supplies this time. A field is written only when the contact still holds the value the
// integration last supplied, or holds nothing at all - anything else is an agent's correction and is
// kept. The returned apply map carries only the fields that actually change, so an unchanged claim
// writes nothing.
//
// next is the record to store back. It follows the claim even for fields that were not applied, so a
// contact an agent later sets back to the supplied value starts following the integration again. An
// empty claim never clears a field, so for those next keeps the previously supplied value and stays
// an honest record of what the contact holds from the integration.
func ResolveExternalSync(current, synced, incoming map[string]string) (apply, next map[string]string) {
	apply, next = map[string]string{}, map[string]string{}
	for _, field := range ExternalSyncFields {
		in := incoming[field]
		if in == "" {
			if last := synced[field]; last != "" {
				next[field] = last
			}
			continue
		}
		next[field] = in
		if cur := current[field]; cur != in && (cur == "" || cur == synced[field]) {
			apply[field] = in
		}
	}
	return apply, next
}
