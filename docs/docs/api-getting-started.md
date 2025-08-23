# API getting started

You can access the Libredesk API to interact with your instance programmatically.

## Generating API keys

1. **Edit agent**: Go to Admin → Teammate → Agent → Edit
2. **Generate new API key**: An API Key and API Secret will be generated for the agent
3. **Save the credentials**: Keep both the API Key and API Secret secure
4. **Key management**: You can revoke / regenerate API keys at any time from the same page

## Using the API

LibreDesk supports two authentication schemes:

### Basic authentication
```bash
curl -X GET "https://your-libredesk-instance.com/api/endpoint" \
  -H "Authorization: Basic <base64_encoded_key:secret>"
```

### Token authentication
```bash
curl -X GET "https://your-libredesk-instance.com/api/endpoint" \
  -H "Authorization: token your_api_key:your_api_secret"
```

## API Documentation

Complete API documentation with available endpoints and examples coming soon.
