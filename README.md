# Xelis CPU Miner (Go)

A high-performance, multi-threaded CPU miner for Xelis cryptocurrency written in Go. This miner implements the XelisHashV2 algorithm and supports the Xelis Stratum protocol.

## Features

- ✅ **XelisHashV2 Algorithm** - Full implementation of the official Xelis hashing algorithm
- 🚀 **Multi-threaded Mining** - Automatically utilizes all CPU cores for maximum performance
- 📊 **Real-time Statistics** - Live hashrate monitoring and share tracking
- 🔄 **Stratum Protocol** - Compatible with standard Xelis mining pools
- 💻 **Cross-platform** - Works on Windows, Linux, and macOS
- 📈 **Efficient Memory Usage** - Optimized scratchpad management per thread

## Performance

Expected hashrate on different CPUs:
- Intel i5 9th Gen (6 cores): ~2.7 KH/s
- Intel i7 10th Gen (8 cores): ~3.5 KH/s
- AMD Ryzen 5 5600X (6 cores): ~4.0 KH/s
- AMD Ryzen 9 5950X (16 cores): ~10 KH/s

*Note: Actual performance may vary depending on your specific hardware configuration.*

## Requirements

- **Go 1.19 or higher**
- **64-bit Operating System**
- **Minimum 2GB RAM** (recommended 4GB+)
- **Xelis wallet address**

## Installation

### Step 1: Install Go

Download and install Go from the official website: https://go.dev/download/

Verify installation:
```bash
go version
```

### Step 2: Clone the Repository

```bash
git clone https://github.com/Gokyona/BernMiner-xelis.git
cd xelis-miner-go
```

### Step 3: Install Dependencies

```bash
go mod init xelis-miner
go get github.com/xelpool/xelishash
go get golang.org/x/crypto/argon2
```

### Step 4: Build the Miner

**Windows:**
```bash
go build -o xelis-miner.exe main.go
```

**Linux/macOS:**
```bash
go build -o xelis-miner main.go
```

## Usage

### Quick Start

Run the miner:
```bash
./xelis-miner
```

The program will prompt you for:
1. **Xelis wallet address** - Your Xelis wallet address (starts with `xel:`)
2. **Miner name** - A unique name for your miner worker

### Example

```
=================================
  Xelis Miner v2.0
  Using XelisHashV2
=================================
Enter your Xelis wallet address: xel:arn8s5988jddpdp2rty5rj87drhv5md0a6hhc62ffeekjm53ga3sqx42dj9
Enter your miner name: myrig01

[*] Connecting to pool...
[*] Starting miner with 6 threads
[*] Subscribe: {"id":0,"error":null,"result":["...","...",32,"..."]}
[*] Authorize: {"id":1,"result":true}
[*] Waiting for jobs...
[*] Difficulty: 1000000
[*] New Job: 0
[Stats] Hashrate: 2756.34 H/s | Shares: 1 (A:1 R:0)
[✓] Share ACCEPTED!
```

## Configuration

### Default Pool

The miner is configured to use:
- **Pool**: `de.xelis.herominers.com:1225` (HeroMiners EU)

### Changing the Pool

Edit `main.go` and modify the pool address in the `main()` function:

```go
client, err := NewStratumClient("your-pool-address:port")
```

### Popular Xelis Pools

- **HeroMiners EU**: `de.xelis.herominers.com:1225`
- **HeroMiners US**: `us.xelis.herominers.com:1225`
- **HeroMiners Asia**: `sg.xelis.herominers.com:1225`

## Understanding the Output

### Statistics Line
```
[Stats] Hashrate: 2756.34 H/s | Shares: 1 (A:1 R:0)
```
- **Hashrate**: Current hashing speed (hashes per second)
- **Shares**: Total shares found
- **A**: Accepted shares
- **R**: Rejected shares

### Share Submission
```
[Thread 2] Found valid share!
  Nonce: 6d6f3d42a385947d
  Hash: a1b2c3d4...
[✓] Share ACCEPTED!
```

## Troubleshooting

### Common Issues

**1. "Failed to connect pool"**
- Check your internet connection
- Verify the pool address and port
- Try a different pool server

**2. "Invalid share" / Shares rejected**
- This is usually temporary - the miner will continue working
- Ensure you're using the latest version
- Check if your system time is synchronized

**3. Low hashrate**
- Close other CPU-intensive applications
- Ensure your CPU is not thermally throttled
- Check CPU power settings (disable power saving mode)

**4. Build errors**
- Ensure Go 1.19+ is installed
- Run `go mod tidy` to clean up dependencies
- Check that all dependencies are properly installed

## Advanced Configuration

### Adjusting Thread Count

By default, the miner uses all available CPU cores. To use fewer cores, modify this line in `main.go`:

```go
numThreads := runtime.NumCPU()  // Change to desired number, e.g., 4
```

### Memory Optimization

Each thread requires approximately 440KB of memory for the scratchpad. If you experience memory issues, reduce the thread count.

## Development

### Project Structure

```
xelis-miner-go/
├── main.go           # Main program file
├── go.mod            # Go module dependencies
├── go.sum            # Dependency checksums
└── README.md         # This file
```

### Dependencies

- `github.com/xelpool/xelishash` - Official XelisHashV2 implementation
- `golang.org/x/crypto/argon2` - Cryptographic utilities

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Areas for Improvement

- [ ] Configuration file support
- [ ] Multiple pool failover
- [ ] Web dashboard
- [ ] Docker support
- [ ] Benchmark mode

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This software is provided "as is", without warranty of any kind. Mining cryptocurrency requires computational resources and electricity. Always ensure you comply with local regulations and laws regarding cryptocurrency mining.

## Acknowledgments

- Xelis Team for the XelisHashV2 algorithm
- xelpool for the Go implementation of XelisHash
- The Xelis community for testing and feedback

## Support

- **Xelis Official Website**: https://xelis.io
- **Xelis Discord**: https://discord.gg/xelis
- **Issues**: Please report bugs via GitHub Issues

## Donations

If you find this miner useful, consider supporting development:

**Xelis**: xel:arn8s5988jddpdp2rty5rj87drhv5md0a6hhc62ffeekjm53ga3sqx42dj9

---

**Happy Mining! ⛏️💎**
