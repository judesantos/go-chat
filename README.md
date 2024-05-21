# GoChat Service

GoChat is a chat service developed using Go that allows users to communicate in real-time over the internet.

The chat server is a WebSocket-based service designed with a channels-for-multiple-connections architecture to ensure higher throughput and stability under high load conditions.


DOT:

- Subscriber - The chat user/client
- Session - A subscriber’s active connection to the chat service.
- Channel - A chat room that can be configured to operate as either a public or private space.
- Server - Service that manages the creation, teardown of sessions and channels. Also manages subscription and sign-in requests


## Benchmark tests:

Tests ran on **apple M2Pro 16GB**

Client and server are both hosted on the same machine.

### Test case

- Create session
- Send a channel join request
- Send a message to channel
- Send a leave channel request
- Teardown session

### Test Client:

Running 1500 test cases per worker, with 20 parallel workers simultaneously sending requests to server.

### Test case transactions:

- Establish session
- Send channel join request
- Send channel message
- Send leave channel request
- Disconnect from server

### Avg. benchmark test duration: 

29 seconds

### Avg. latency per test case:

**0.967** millisecond per test case = 29 seconds / 30,000 test cases

## Features

- Real-time messaging
- Support for multiple chat rooms
- User authentication
- Simple and lightweight
- Customizable and extendable

## Technologies Used

- Go (Golang)
- WebSocket (gorilla/websocket)
- HTTP Router (gorilla/mux)
- JSON Web Tokens (jwt-go)
- Sqlite
- Redis

## Installation

### Prerequisites

- Go installed on your machine. You can download and install Go from [here](https://golang.org/dl/).
- sqlite

### Clone the Repository

```bash
git clone https://github.com/judesantos/go-chat.git
cd go-chat
```

### Setup Environment Variables

Create a .env file in the project root directory and add the following environment variables:

```bash
ENV=development

SERVER_HOST=server_domain_url
SERVER_PORT=server_port
SERVER_DB=path_to_sqlite_db_file

PUBSUB_SERVER_HOST=redis_server_domain_url
PUBSUB_SERVER_PORT=redis_server_port
PUBSUB_SERVER_PASS=redis_db_password

LOG_OUTPUT=stdout,file    [ Log to file (file), or terminal console (stdout) ]
LOG_FILE=logs/server.log  [ Log file path and file name ]
LOG_CONSOLE_LEVEL=trace   [ Min. log level for console logs  ]
LOG_FILE_LEVEL=info       [ Min. log level for file logs ]
```
### Install Dependencies

```bash
go mod tidy
```

### Build and Run

```bash
go build -o chat-server server/main.go
./chat-server
```

### Test - build server.test and run

Make sure chat-server is running from the previous step.

```bash
go build -o server_test test/server-test.go
./server_test
```

To run repeatedly in parallel using workers

```bash
./test -r 1000 -w 10
```
Will run 10,000 tests; 1000 synchronous tests (-r) for each go-routine worker, with 10 workers in parallel with other workers (-w). 

## Usage
Once the chat service is running, users can connect using a WebSocket client or a chat client that supports WebSocket connections.

## API Endpoints
- POST /signup - Register a new user
- POST /login - Login and obtain a JWT token
- GET /ws?name=username - Connect to the chat service using WebSocket

## Database Setup
- Create a sqlite server. Schema will automatically be created on setup.
- Update the database connection details in the .env file.


## JWT Secret Key



## Contributing

Contributions are welcome! Please follow these steps to contribute:

1. Fork the repository.
2. Create a new branch (git checkout -b feature-branch).
3. Make your changes.
4. Commit your changes (git commit -m 'Add some feature').
5. Push to the branch (git push origin feature-branch).
6. Open a Pull Request.

Please ensure your code adheres to the project's coding standards and includes appropriate tests.

## License
This project is licensed under the MIT License - see the LICENSE file for details.

## Contact
If you have any questions or feedback, feel free to reach out:

- [yourtechy]jude@yourtechy.com
- [Jude santos]jude.msantos@gmail.com
