# Install libredesk

Docker Compose runs libredesk, PostgreSQL, and Redis. Install Docker with the Compose plugin first.

## Try it locally

Download `setup.sh` from a published release into an empty directory, then run:

```sh
sh setup.sh local
```

Setup downloads the configuration and Compose file from the same release as the Docker image. It generates database credentials and an encryption key, creates `uploads`, and starts the services. Configuration files are readable only by their owner.

Open http://localhost:9000/setup. Copy the temporary setup code from the terminal, then choose your admin email and password. The code expires after one hour. Restarting the app prints a new code:

```sh
docker compose restart app
docker compose logs --tail=30 app
```

Each code works once. Existing installations and installations that supply `LIBREDESK_SYSTEM_USER_PASSWORD` keep their existing login flow.

Running setup again preserves configuration, passwords, and uploaded files. Keep the existing files together. Setup refuses to overwrite a partially configured installation.

## Run on your server

Point your domain at the server. In an empty directory containing `setup.sh`, run:

```sh
sh setup.sh production https://support.example.com
```

The app listens on the server's loopback address at port 9000. PostgreSQL and Redis use automatically assigned ports bound to loopback. Existing Compose installations keep their previous default ports.

Install nginx and Certbot using your operating system's packages. Save this nginx configuration as `/etc/nginx/sites-available/libredesk`, replacing the domain:

```nginx
server {
    listen 80;
    server_name support.example.com;
    client_max_body_size 100M;

    location / {
        proxy_pass http://127.0.0.1:9000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Client-IP $remote_addr;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

On Debian or Ubuntu, enable it and request your TLS certificate:

```sh
sudo ln -s /etc/nginx/sites-available/libredesk /etc/nginx/sites-enabled/libredesk
sudo nginx -t
sudo systemctl reload nginx
sudo certbot --nginx -d support.example.com
```

Allow HTTP and HTTPS traffic through your firewall. Open your HTTPS address followed by `/setup`, then create your admin account with the code from the app logs. Keep secure cookies enabled.

If you already use a reverse proxy, forward HTTP and WebSocket traffic to port 9000 and set `X-Client-IP` to the client address.

## Finish setup

After signing in, the conversation screen shows your setup checklist:

1. Confirm your public URL and timezone under Admin, General, then save.
2. Create your first email or livechat inbox.
3. Configure notification email for account invitations and password resets, or skip it until later.
4. Send an email to your inbox or a message through your installed website widget. Reply from libredesk, then check progress.

The conversation step completes after an incoming message receives a successfully sent public reply in the same conversation. Private notes and failed replies do not complete it. Teammate invitations are optional.

## Check an installation

The diagnostic command checks configuration, PostgreSQL, Redis, local upload permissions, and cookie settings. It does not start the server or change your database.

```sh
docker compose exec app ./libredesk --check
```

For a release binary:

```sh
./libredesk --check
```

## Install a release binary

Extract the archive and copy `config.sample.toml` to `config.toml`. Set your PostgreSQL and Redis connection details. Generate `app.encryption_key` once:

```sh
openssl rand -hex 16
```

Create the upload folder, install the schema, check dependencies, and start libredesk:

```sh
mkdir -p uploads
./libredesk --install
./libredesk --upgrade --yes
./libredesk --check
./libredesk
```

Open `/setup` and use the temporary code printed on startup. Production requires the HTTPS reverse proxy described above.

## Preserve data during upgrades

Back up PostgreSQL, `uploads`, `config.toml`, and `.env`. The encryption key is required to read stored inbox passwords. Keep it unchanged.

`--install` leaves an installed database unchanged. It refuses a partially installed database containing tables. `--upgrade` applies database migrations.

Erasing data requires the explicit `--install --reset-database` combination and confirmation. The `--yes` flag alone never authorizes erasing an existing database.

The installer pins `LIBREDESK_IMAGE` in `.env`. When upgrading, download the configuration examples from the target release, retain your existing secrets, and update the image tag deliberately. Check that release's notes before starting it.
