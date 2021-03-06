// yibc project main.go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

var (
	//flags
	address = flag.String("ip", fmt.Sprintf("%s:%s", GetIpAddress()[0], BLOCKCHAIN_PORT), "Public facing ip address")
	self    = struct {
		*Keypair
		*Blockchain
		*Network
	}{}
)

func init() {
	flag.Parse()
}

func main() {
	//Setup keys
	keypair, _ := OpenConfiguration(HOME_DIRECTORY_CONFIG)
	if keypair == nil {
		fmt.Println("生成密钥对。。。。")
		keypair = GenerateNewKeypair()
		WriteConfiguration(HOME_DIRECTORY_CONFIG, keypair)
	}
	self.Keypair = keypair

	//Setup Network
	self.Network = SetupNetwork(*address, BLOCKCHAIN_PORT)
	go self.Network.Run()
	for _, n := range SEED_NODES() {
		self.Network.ConnectionsQueue <- n
	}

	//Setup blockchain
	self.Blockchain = SetupBlockChain()
	go self.Blockchain.Run()

	//Read Stdin to create transations
	stdin := ReadStdin()
	for {
		select {
		case str := <-stdin:
			self.Blockchain.TransactionsQueue <- CreateTransaction(str)
		case msg := <-self.Network.IncomingMessages:
			HandleIncomingMessage(msg)
		}
	}
}

func CreateTransaction(txt string) *Transaction {
	t := NewTransaction(self.Keypair.Public, nil, []byte(txt))
	t.Header.Nonce = t.GenerateNonce(TRANSACTION_POW)
	t.Signature = t.Sign(self.Keypair)
	return t
}

//处理传入信息：交易信息和区块信息
func HandleIncomingMessage(msg Message) {
	switch msg.Identifier {
	case MESSAGE_SEND_TRANSACTION:
		t := new(Transaction)
		_, err := t.UnmarshalBinary(msg.Data)
		if err != nil {
			networkError(err)
			break
		}
		self.Blockchain.TransactionsQueue <- t
	case MESSAGE_SEND_BLOCK:
		b := new(Block)
		err := b.UnmarshalBinary(msg.Data)
		if err != nil {
			networkError(err)
			break
		}
		self.Blockchain.BlocksQueue <- *b
	}
}

func ReadStdin() chan string {

	cb := make(chan string)
	sc := bufio.NewScanner(os.Stdin)

	go func() {
		for sc.Scan() {
			cb <- sc.Text()
		}
	}()

	return cb

}
