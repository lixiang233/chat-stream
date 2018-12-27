package main

import (
	"bufio"
	"fmt"
	"google.golang.org/grpc"
	"io"
	"net"
	"os"
	"time"
	"context"
	pb "github.com/chat-stream/server/api"
)

func main()  {
	conn, err := grpc.Dial("/var/run/stream_test.sock",
		grpc.WithInsecure(),
		grpc.WithDialer(func(s string, duration time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", s, duration)
	}))
	if err != nil{
		fmt.Println(err)
	}
	defer conn.Close()
	NewSessions(conn)

}

func NewSessions(conn *grpc.ClientConn)  {
	client := pb.NewStreamServiceClient(conn)
	//ctx, cancle := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
	ctx, cancle := context.WithCancel(context.Background())
	stream, err := client.Communicate(ctx)
	if err != nil {
		fmt.Println(err)
	}

	if stream == nil{
		fmt.Println("nil stream, stop!")
		return
	}

	inputChan := make(chan string, 5)
	go func() {
		inp := bufio.NewReader(os.Stdin)
		for {
			//fmt.Print("You:")
			str, _:= inp.ReadString('\n')
			inputChan <- str
		}
	}()

	go func() {
		for {
			select {
			case str := <- inputChan:
				err := stream.Send(&pb.StreamRequest{
					Name:"client",
					Message:str,
				})
				if err != nil {
					fmt.Printf("send err, %v", err)
				}else {
					//fmt.Println("send to server success")
					fmt.Print("YOU:", str)
				}
			case <- time.After(time.Duration(time.Second * 300)):
				fmt.Println("no message after 300s, call cancle")
				cancle()
			}
		}
	}()

	for {
		message, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("server stop grpc")
			return
		}
		if err != nil {
			fmt.Println(err)
			fmt.Println("Stop grpc client")
			return
		}
		fmt.Printf("%s : %s", message.Name, message.Message)
	}
}
