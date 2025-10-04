export function computeRecipientsFromMessage (message, contactEmail, inboxEmail) {
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

    // Dedup + remove inbox email
    const clean = list =>
        Array.from(new Set(list.filter(email => email && email !== inboxEmail)))

    return {
        to: clean(toList),
        cc: clean(ccList),
        // BCC stays empty user is supposed to add it manually.
        bcc: [],
    }
}
