# Installation

Libredesk is a single binary application that requires postgres and redis to run. You can install it using the binary or docker.

## Binary

1. Download the [latest release](https://github.com/abhinavxd/libredesk/releases) and extract the libredesk binary.
2. `./libredesk --install` to install the tables in the Postgres DB (⩾ 13) and set the System user password.
3. Run `./libredesk` and visit `http://localhost:9000` and login with the email `System` and the password you set during installation.

!!! Tip
    To set the System user password during installation, set the environment variables:
    `LIBREDESK_SYSTEM_USER_PASSWORD=xxxxxxxxxxx ./libredesk --install`


## Docker

The latest image is available on DockerHub at `libredesk/llibredeskistmonk:latest`

The recommended method is to download the [docker-compose.yml](https://github.com/abhinavxd/libredesk/blob/master/docker-compose.yml) file, customize it for your environment and then to simply run `docker compose up -d`.

```shell
# Download the compose file to the current directory.
curl -LO https://github.com/abhinavxd/libredesk/raw/master/docker-compose.yml

# Copy the config.sample.toml to config.toml and edit it as needed.
cp config.sample.toml config.toml

# Run the services in the background.
docker compose up -d

# Setting System user password.
docker exec -it libredesk_app ./libredesk --set-system-user-password
```

Go to `http://localhost:9000` and login with the email `System` and the password you set using the `--set-system-user-password` command.

---


## Compiling from source

To compile the latest unreleased version (`master` branch):

1. Make sure `go`, `nodejs`, and `pnpm` are installed on your system.
2. `git clone git@github.com:abhinavxd/libredesk.git`
3. `cd libredesk && make`. This will generate the `libredesk` binary.
