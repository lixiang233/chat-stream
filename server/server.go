package main

import (
	"bufio"
	"errors"
	"fmt"
	pb "github.com/chat-stream/server/api"
	"google.golang.org/grpc"
	"io"
	"net"
	"os"
	"time"
)

const STREAM_PATH = "/var/run/stream_test.sock"

type StreamService struct {
	ClientInfo map[string]string
	recChan chan *pb.StreamRequest
	sendChan chan *pb.StreamResponse
}

func (s *StreamService)Communicate(stream pb.StreamService_CommunicateServer) error {
	fmt.Println("A new client online")
	ctx := stream.Context()
	stopChan := make(chan int, 1)

	// start routine rec done message
	go func() {
		for {
			select {
			case <-ctx.Done():
				stopChan <- 1
				fmt.Println("recv ctx.Done")
			case <-time.After(time.Duration(time.Second * 3)):
				stopChan <- 0
				//fmt.Println("go func send health check")
			}
		}
	}()

	//recChan := make(chan *pb.StreamRequest, 100)
	//sendChan := make(chan *pb.StreamResponse, 100)
	// recive message and put it to channel
	go func() {
		for {
			message, err := stream.Recv()
			if err == io.EOF {
				stopChan <- 2
				fmt.Println("client send over")
				return
			}
			if err != nil {
				stopChan <- 3
				fmt.Println("rec error", err)
				return
			}
			s.recChan <- message
		}
	}()
	// read from channel, response to client
	go func() {
		for {
			message := <- s.sendChan
			err := stream.Send(message)
			if err != nil {
				fmt.Println("send err ", err)
			}
		}
	}()
	// generate response from request
	go func() {
		for  {
			message := <- s.recChan
			fmt.Printf("%s : %s", message.Name, message.Message)
			s.sendChan <- &pb.StreamResponse{
				Name:"SERVER",
				Message:fmt.Sprintf("I have recived your message, first cha is %s \n", message.Message[0]),
			}
		}
	}()

	go func() {
		inp := bufio.NewReader(os.Stdin)
		for {
			//fmt.Print("You:")
			str, _:= inp.ReadString('\n')
			s.sendChan <- &pb.StreamResponse{
				Name:"SERVER",
				Message:str,
			}
		}
	}()

	for {
		select {
		case mes := <- stopChan:
			switch mes {
			case 0:
				//fmt.Println("rec healthy check")
			case 1:
				fmt.Println("client cancle grpc")
				return errors.New("EOF")
			case 2:
				fmt.Println("client send mes ovr")
				return nil
			case 3:
				fmt.Println("program err")
				return errors.New("Por Error!")
			}
		case <-time.After(time.Duration(6 * time.Second)):
			fmt.Println("healthy check not rcv!")
			return errors.New("server healthy check not rcv!")
		}
	}
}

func main()  {

	err := os.Remove(STREAM_PATH)
	err = os.MkdirAll("/var/run", 0700)
	if err != nil {
		fmt.Printf("OS path error %v", err)
	}

	conn, err := net.ListenUnix("unix", &net.UnixAddr{Name: STREAM_PATH, Net: "unix"})
	s := grpc.NewServer()
	pb.RegisterStreamServiceServer(s, &StreamService{ClientInfo:make(map[string]string),
		recChan: make(chan *pb.StreamRequest, 100),
		sendChan: make(chan *pb.StreamResponse, 100),
	})

	fmt.Println("start listen")
	err = s.Serve(conn)

	if err != nil{
		fmt.Println(err)
	}
	err = conn.Close()
	if err != nil {
		fmt.Println(err)
	}
}
