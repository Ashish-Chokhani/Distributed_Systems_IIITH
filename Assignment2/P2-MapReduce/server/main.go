package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
  "time"
	"path/filepath"
	"strings"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
)

type Worker struct {
	pb.UnimplementedWorkerServer
  grpcServer *grpc.Server
}

func (w *Worker) Map(ctx context.Context, req *pb.MapRequest) (*pb.TaskResponse, error) {
	file, err := os.Open(req.Filename)
	if err != nil {
		return &pb.TaskResponse{Success: false, Message: "Failed to open file"}, err
	}
	defer file.Close()

	for i := 0; i < int(req.NumReducers); i++ {
		f, _ := os.Create(fmt.Sprintf("intermediate/mr-%d-%d.txt", req.MapTaskId, i))
		defer f.Close()
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words := strings.Fields(scanner.Text())
		for _, word := range words {
			bucket := hash(word) % int(req.NumReducers)
			f, _ := os.OpenFile(fmt.Sprintf("intermediate/mr-%d-%d.txt", req.MapTaskId, bucket), os.O_APPEND|os.O_WRONLY, 0644)
			if req.Mode == "inverted_index" {
				fmt.Fprintf(f, "%s %s\n", word, filepath.Base(req.Filename))
			} else if req.Mode == "word_count" {
				fmt.Fprintf(f, "%s 1\n", word)
			}
			f.Close()
		}
	}

  // w.grpcServer.GracefulStop()
	return &pb.TaskResponse{Success: true, Message: "Map task completed"}, nil
}

func (w *Worker) Reduce(ctx context.Context, req *pb.ReduceRequest) (*pb.TaskResponse, error) {
	if req.Mode == "inverted_index" {
		return w.reduceInvertedIndex(req)
	} else if req.Mode == "word_count" {
		return w.reduceWordCount(req)
	}
  // w.grpcServer.GracefulStop()
	return &pb.TaskResponse{Success: false, Message: "Invalid mode"}, nil
}

func (w *Worker) reduceInvertedIndex(req *pb.ReduceRequest) (*pb.TaskResponse, error) {
	invertedIndex := make(map[string]map[string]bool)

	for mapTaskId := 0; ; mapTaskId++ {
    fmt.Printf("Iterating %d",mapTaskId)
		filename := fmt.Sprintf("intermediate/mr-%d-%d.txt", mapTaskId, req.ReduceTaskId)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			break
		}

		file, _ := os.Open(filename)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var word, filename string
			fmt.Sscanf(scanner.Text(), "%s %s", &word, &filename)
			if invertedIndex[word] == nil {
				invertedIndex[word] = make(map[string]bool)
			}
			invertedIndex[word][filename] = true
		}
		file.Close()
	}

	outputFile, _ := os.Create(fmt.Sprintf("output/out-%d.txt", req.ReduceTaskId))
	defer outputFile.Close()

	for word, files := range invertedIndex {
		fileList := []string{}
		for file := range files {
			fileList = append(fileList, file)
		}
		fmt.Fprintf(outputFile, "%s %v\n", word, fileList)
	}

	resp := &pb.TaskResponse{Success: true, Message: "Reduce task completed"}
  time.Sleep(2 * time.Second) // Give master time to receive response
  return resp, nil
}

func (w *Worker) reduceWordCount(req *pb.ReduceRequest) (*pb.TaskResponse, error) {
	wordCount := make(map[string]int)

	for mapTaskId := 0; ; mapTaskId++ {
		filename := fmt.Sprintf("intermediate/mr-%d-%d.txt", mapTaskId, req.ReduceTaskId)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			break
		}

		file, _ := os.Open(filename)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var word string
			var count int
			fmt.Sscanf(scanner.Text(), "%s %d", &word, &count)
			wordCount[word] += count
		}
		file.Close()
	}

	outputFile, _ := os.Create(fmt.Sprintf("output/out-%d.txt", req.ReduceTaskId))
	defer outputFile.Close()

	for word, count := range wordCount {
		fmt.Fprintf(outputFile, "%s %d\n", word, count)
	}
	return &pb.TaskResponse{Success: true, Message: "Reduce task completed"}, nil
}

func hash(s string) int {
	h := 0
	for _, c := range s {
		h = 31*h + int(c)
	}
	return h
}

func main() {
  os.MkdirAll("intermediate", os.ModePerm)
  os.MkdirAll("output", os.ModePerm)
	port := os.Args[1]
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		fmt.Printf("Failed to listen on port %s: %v\n", port, err)
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterWorkerServer(grpcServer, &Worker{})

	fmt.Printf("Worker server listening on :%s\n", port)
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}
