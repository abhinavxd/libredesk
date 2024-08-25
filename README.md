# Artemis

- Open source customer support desk.


- Haven't decided name for the project yet, it's in very early stage.

## Dev setup

### frontend
1. [Install bun](https://bun.sh/docs/installation)
2. run `make build-frontend`


### 2. backend
1. run `docker compose up`
2. run `cp config.sample.toml config.toml`
3. run `make`
4. run `artemis.bin --install`
5. run `artemis.bin`