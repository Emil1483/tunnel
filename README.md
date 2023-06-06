# Tunnel

This is a Go project that allows you to expose a locally running webserver to the internet without the need for port forwarding. It does this by acting as a proxy between the internet and your locally running webserver.

### Project Structure

The project is organized into two folders: client and server.

#### Client

The client is responsible for making HTTP requests on behalf of the server and forwarding the responses back to the server.

#### Server

The server exposes a websocket endpoint (/ws) and handles all other HTTP requests at the root endpoint (/). The server receives HTTP requests from the internet, sends the request data to the connected client via the websocket, and waits for the response. It then responds to the HTTP request with the response by your locally running webserver.

---

### Usage

Deploy the server on any web service provider. I recommend using [render.com](https://render.com/). Remember to set the `ACCESS_TOKEN` environment variable.

##### To run the client

First, install Go if you haven't already: [Go Installation Guide](https://go.dev/doc/install). Then,

```bash
git clone https://github.com/Emil1483/tunnel
cd tunnel/client
```

**Build the client the client with**

```bash
go build -o tunnel
```

**Create a settings.conf file** In the same directory as the executable you just created

```json
{
  "tunnelUri": "wss://YOUR_DEPLOYED_SERVER_URI/ws",
  "endpointUri": "http://localhost:3000",
  "accessToken": "YOUR_ACCESS_TOKEN"
}
```

**Run the client with**

```bash
./tunnel
```

Optionally, you can add the directory of your binary and `settings.conf` files to your path environment variable to easily access your tunnel from anywhere in the terminal.
