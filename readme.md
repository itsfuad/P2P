# P2P Mesh Network

This repository contains the source code for a P2P (Peer-to-Peer) Mesh Network application. The project is designed to facilitate file sharing and peer discovery in a decentralized manner.

## Features

- **Peer Discovery**: Automatically discover and connect to peers in the network.
- **File Sharing**: Share files with connected peers.
- **Web UI**: Manage peers and files through a web-based user interface.
- **Encryption**: Secure file transfers using RSA encryption.

## Project Structure

- `internal/crypto`: Contains encryption-related code.
- `internal/dht`: Implements the Distributed Hash Table (DHT) for peer discovery.
- `internal/node`: Core logic for managing peers and files.
- `internal/transfer`: Handles file chunking and transfer.
- `internal/webui`: Web UI for managing the network.
- `main.go`: Entry point for the application.

## Getting Started

### Prerequisites

- Go 1.23.2 or later

### Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/itsfuad/P2P.git
    cd P2P
    ```

2. Build the project:
    ```sh
    go build -o p2p
    ```

3. Run the application:
    ```sh
    ./p2p -port 3000 -webui 8080
    ```

### Running Tests

To run the tests, use the following command:
```sh
go test ./...
```

## Usage

1. Start the application using the command mentioned in the installation section.
2. Open your browser and navigate to `http://localhost:8080` to access the Web UI.
3. Use the Web UI to manage peers and share files.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.

## License

This project is licensed under  [LICENSE](LICENSE) file for details.

## Acknowledgements

- The Go Authors for the Go programming language.
- Open-source community for various libraries and tools.
