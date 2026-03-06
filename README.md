# Axype CLI

a CLI for the Axype pasteloader

## Supports

- [x] Publishing pastes
- [x] Creating projects

## Requirements

- Go lang (building)

## Building

```sh
git clone git@github.com:axype/cli.git cli
cd cli
go mod download
```

```sh
go build -ldflags="-s -w" -o=dist/axype .
# ^ creates the binary inside the dist folder
```

you can place the built executable in your PATH/bin

```sh
# Unix example:

cp dist/axype $(go env GOPATH)/bin/axype
```

## Usage

```sh
# authentication:
axype set-token <TOKEN>
# -| change <TOKEN> to your Axype TOKEN
# -| it stores the authentication file under $HOME/.axype/secret

# deleting authentication file:
axype clear-token

# creating a new project:
axype init

# publishing paste (make sure the axype paste is built):
axype publish <PASTE> [INPUT]
# -| INPUT is optional

# removing .axype folder:
axype clear
# -| ^ you still need to delete the binary yourself
```
