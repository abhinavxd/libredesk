# Livechat widget JWT test page

A single-file page for testing the livechat widget's JWT contact authentication
locally. Signs an HS256 JWT in the browser from an editable payload and loads the
widget with it, or as an anonymous visitor with the JWT checkbox off.

## Run

Serve the file over HTTP (the widget won't load from `file://`):

```
python -m http.server 8005
```

Open `http://localhost:8005`, then fill in:

- **baseURL**: your Libredesk instance (default `http://localhost:8001` - Default points to the widget dev server, not the API server)
- **inboxID**: the livechat inbox's UUID (from the inbox's installation snippet)
- **Signing secret**: the "Secret Key" from that inbox's Security tab

`exp` is set to now + 1 hour on every generate. Use "Clear session" to drop the
widget's cookies and start over as a brand-new visitor.
