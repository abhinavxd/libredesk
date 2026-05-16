// Strip +conv-{uuid-v4} from email if present.
// Only matches strict UUID v4 format (36 chars)
// e.g., support+conv-13216cf7-6626-4b0d-a938-46ce65a20701@domain.com -> support@domain.com
export function stripConvUUID (email) {
    if (!email) return email
    return email.replace(/\+conv-[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[a-f0-9]{4}-[a-f0-9]{12}@/i, '@')
}

// Extract bare address from "Name <addr@domain>", else return input trimmed.
export function extractEmail (email) {
    if (!email) return email
    const m = email.match(/<([^>]+)>/)
    return (m ? m[1] : email).trim()
}

// inboxAddresses can be a single string or an array of inbox-owned addresses
// (e.g. every email inbox's from + reply_to) to dedupe against, so a reply
// never targets any inbox this Libredesk instance owns.
export function computeRecipientsFromMessage (message, contactEmail, inboxAddresses = []) {
    const meta = message?.meta || {}
    const isIncoming = message.type === 'incoming'

    // Build TO field
    const toList = isIncoming
        ? meta.from && meta.from.length
            ? meta.from
            : contactEmail
                ? [contactEmail]
                : []
        : meta.to && meta.to.length
            ? meta.to
            : contactEmail
                ? [contactEmail]
                : []

    // Build CC field
    let ccList = meta.cc || []

    if (isIncoming) {
        // Include original 'to' recipients in CC to preserve full thread context (e.g. other participants)
        if (Array.isArray(meta.to))
            ccList = ccList.concat(meta.to)

        // If someone else replies (not the original contact), re-add original contact to CC to keep them in the loop.
        if (
            contactEmail &&
            !toList.includes(contactEmail) &&
            !ccList.includes(contactEmail)
        ) {
            ccList.push(contactEmail)
        }
    }

    const normalize = e => stripConvUUID(extractEmail(e)).toLowerCase()
    const dedupeSet = new Set(
        (Array.isArray(inboxAddresses) ? inboxAddresses : [inboxAddresses])
            .filter(Boolean)
            .map(normalize)
    )
    const clean = list =>
        Array.from(new Set(list.filter(email => email && !dedupeSet.has(normalize(email)))))

    return {
        to: clean(toList),
        cc: clean(ccList),
        // BCC stays empty user is supposed to add it manually.
        bcc: [],
    }
}
