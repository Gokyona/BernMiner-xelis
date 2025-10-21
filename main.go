package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xelpool/xelishash"
)

// -------------------- Stratum 客户端 --------------------

type StratumClient struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
}

func NewStratumClient(addr string) (*StratumClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &StratumClient{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

func (s *StratumClient) Send(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.conn.Write([]byte(data + "\n"))
	if err != nil {
		log.Println("Send error:", err)
	}
	return err
}

func (s *StratumClient) ReadLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return strings.TrimSpace(line), nil
		}
		return strings.TrimSpace(line), err
	}
	return strings.TrimSpace(line), nil
}

// -------------------- 挖矿统计 --------------------

type MiningStats struct {
	hashCount   uint64
	shareFound  uint64
	shareAccept uint64
	shareReject uint64
	startTime   time.Time
}

func (m *MiningStats) AddHash(count uint64) {
	atomic.AddUint64(&m.hashCount, count)
}

func (m *MiningStats) AddShare() {
	atomic.AddUint64(&m.shareFound, 1)
}

func (m *MiningStats) AddAccept() {
	atomic.AddUint64(&m.shareAccept, 1)
}

func (m *MiningStats) AddReject() {
	atomic.AddUint64(&m.shareReject, 1)
}

func (m *MiningStats) GetHashrate() float64 {
	elapsed := time.Since(m.startTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(atomic.LoadUint64(&m.hashCount)) / elapsed
}

func (m *MiningStats) PrintStats() {
	hashrate := m.GetHashrate()
	shares := atomic.LoadUint64(&m.shareFound)
	accepts := atomic.LoadUint64(&m.shareAccept)
	rejects := atomic.LoadUint64(&m.shareReject)
	
	fmt.Printf("\r[Stats] Hashrate: %.2f H/s | Shares: %d (A:%d R:%d)    ",
		hashrate, shares, accepts, rejects)
}

// -------------------- Mining --------------------

type MiningJob struct {
	jobID         string
	timestamp     []byte
	headerHash    []byte
	target        *big.Int
}

type SessionInfo struct {
	extraNonce []byte
	publicKey  []byte
}

func RunMining(client *StratumClient, wallet, worker string) {
	numThreads := runtime.NumCPU()
	fmt.Printf("[*] Starting miner with %d threads\n", numThreads)

	// Subscribe
	client.Send(`{"id":0,"method":"mining.subscribe","params":["GoBernMiner/2.0",["xel/v2"]]}`)
	resp, _ := client.ReadLine()
	fmt.Println("[*] Subscribe:", resp)

	// 解析 subscribe 响应获取 extraNonce 和 publicKey
	var subscribeResp struct {
		ID     int           `json:"id"`
		Result []interface{} `json:"result"`
	}
	
	session := &SessionInfo{}
	if err := json.Unmarshal([]byte(resp), &subscribeResp); err == nil && len(subscribeResp.Result) >= 4 {
		if extraNonceHex, ok := subscribeResp.Result[1].(string); ok {
			session.extraNonce, _ = hex.DecodeString(extraNonceHex)
		}
		if publicKeyHex, ok := subscribeResp.Result[3].(string); ok {
			session.publicKey, _ = hex.DecodeString(publicKeyHex)
		}
		fmt.Printf("[*] Extra Nonce: %x\n", session.extraNonce)
		fmt.Printf("[*] Public Key: %x\n", session.publicKey)
	}

	// Authorize
	auth := fmt.Sprintf(`{"id":1,"method":"mining.authorize","params":["%s","%s",""]}`, wallet, worker)
	client.Send(auth)
	resp, _ = client.ReadLine()
	fmt.Println("[*] Authorize:", resp)

	fmt.Println("[*] Waiting for jobs...")

	difficulty := big.NewInt(1)
	target := new(big.Int).Lsh(big.NewInt(1), 256)
	target.Div(target, difficulty)

	var cancelMining context.CancelFunc
	var currentJob *MiningJob
	var jobMu sync.Mutex

	stats := &MiningStats{startTime: time.Now()}

	// 统计显示协程
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats.PrintStats()
		}
	}()

	// 响应处理协程
	responseChan := make(chan string, 100)
	go func() {
		for {
			line, err := client.ReadLine()
			if err != nil {
				log.Println("Error reading:", err)
				return
			}
			if len(line) > 0 {
				responseChan <- line
			}
		}
	}()

	// 主循环处理消息
	for line := range responseChan {
		// 处理提交响应
		if strings.Contains(line, `"id":4`) {
			var submitResp struct {
				ID     int         `json:"id"`
				Result interface{} `json:"result"`
				Error  interface{} `json:"error"`
			}
			if err := json.Unmarshal([]byte(line), &submitResp); err == nil {
				if submitResp.Error != nil {
					stats.AddReject()
					fmt.Printf("\n[!] Share REJECTED: %v\n", submitResp.Error)
				} else {
					stats.AddAccept()
					fmt.Printf("\n[✓] Share ACCEPTED!\n")
				}
			}
			continue
		}

		// 难度变化
		if strings.Contains(line, `"method":"mining.set_difficulty"`) {
			var diffMsg struct {
				Method string        `json:"method"`
				Params []interface{} `json:"params"`
			}
			if err := json.Unmarshal([]byte(line), &diffMsg); err == nil && len(diffMsg.Params) > 0 {
				if d, ok := diffMsg.Params[0].(float64); ok {
					difficulty.SetUint64(uint64(d))
					target = new(big.Int).Lsh(big.NewInt(1), 256)
					target.Div(target, difficulty)
					fmt.Printf("\n[*] Difficulty: %v\n", difficulty)
				}
			}
			continue
		}

		// 新任务
		if strings.Contains(line, `"method":"mining.notify"`) {
			var notify struct {
				Method string        `json:"method"`
				Params []interface{} `json:"params"`
			}
			if err := json.Unmarshal([]byte(line), &notify); err != nil || len(notify.Params) < 3 {
				fmt.Println("[!] Invalid notify params")
				continue
			}

			jobID, _ := notify.Params[0].(string)
			timestampHex, _ := notify.Params[1].(string)
			headerHashHex, _ := notify.Params[2].(string)
			
			fmt.Printf("\n[*] New Job: %s\n", jobID)

			// 解码
			timestampBytes, err := hex.DecodeString(timestampHex)
			if err != nil {
				fmt.Println("[!] Invalid timestamp")
				continue
			}
			
			headerBytes, err := hex.DecodeString(headerHashHex)
			if err != nil {
				fmt.Println("[!] Invalid header hash")
				continue
			}

			// 取消旧任务
			if cancelMining != nil {
				cancelMining()
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancelMining = cancel

			// 更新当前任务
			jobMu.Lock()
			currentJob = &MiningJob{
				jobID:      jobID,
				timestamp:  timestampBytes,
				headerHash: headerBytes,
				target:     new(big.Int).Set(target),
			}
			jobMu.Unlock()

			// 启动多线程挖矿
			for t := 0; t < numThreads; t++ {
				go mineWorker(ctx, t, currentJob, session, client, worker, stats)
			}
		}
	}
}

func mineWorker(ctx context.Context, threadID int, job *MiningJob, session *SessionInfo, client *StratumClient, worker string, stats *MiningStats) {
	// 创建可复用的 scratchpad
	var scratchpad xelishash.ScratchPadV2
	
	nonce := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonce, uint64(threadID)<<56)
	
	hashCount := uint64(0)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	nonceCounter := uint64(0)
	
	for {
		select {
		case <-ctx.Done():
			stats.AddHash(hashCount)
			return
		case <-ticker.C:
			stats.AddHash(hashCount)
			hashCount = 0
		default:
			// 递增 nonce
			nonceCounter++
			binary.LittleEndian.PutUint64(nonce, (uint64(threadID)<<56)|nonceCounter)
			
			// 构造 MinerWork 结构 (112 bytes)
			// 0-31: Header work hash (32 bytes)
			// 32-39: Timestamp (8 bytes)
			// 40-47: Nonce (8 bytes)
			// 48-79: Extra nonce (32 bytes)
			// 80-111: Public key (32 bytes)
			minerWork := make([]byte, 112)
			copy(minerWork[0:32], job.headerHash)
			copy(minerWork[32:40], job.timestamp)
			copy(minerWork[40:48], nonce)
			copy(minerWork[48:80], session.extraNonce)
			copy(minerWork[80:112], session.publicKey)
			
			// 计算 XelisHashV2
			hash := xelishash.XelisHashV2(minerWork, &scratchpad)
			hashInt := new(big.Int).SetBytes(hash[:])
			hashCount++

			// 检查是否满足难度
			if hashInt.Cmp(job.target) <= 0 {
				stats.AddShare()
				
				nonceHex := hex.EncodeToString(nonce)
				
				submitData := map[string]interface{}{
					"id":     4,
					"method": "mining.submit",
					"params": []string{worker, job.jobID, nonceHex},
				}
				submitJSON, _ := json.Marshal(submitData)
				
				fmt.Printf("\n[Thread %d] Found valid share!\n", threadID)
				fmt.Printf("  Nonce: %s\n", nonceHex)
				fmt.Printf("  Hash: %x...\n", hash[:8])
				
				client.Send(string(submitJSON))
			}
		}
	}
}

// -------------------- Main --------------------

func main() {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println("=================================")
	fmt.Println("  Xelis Miner v2.0")
	fmt.Println("  Using XelisHashV2")
	fmt.Println("=================================")
	
	fmt.Print("Enter your Xelis wallet address: ")
	wallet, _ := reader.ReadString('\n')
	wallet = strings.TrimSpace(wallet)

	fmt.Print("Enter your miner name: ")
	worker, _ := reader.ReadString('\n')
	worker = strings.TrimSpace(worker)

	fmt.Println("\n[*] Connecting to pool...")
	client, err := NewStratumClient("de.xelis.herominers.com:1225")
	if err != nil {
		log.Fatal("Failed to connect pool:", err)
	}

	RunMining(client, wallet, worker)
}