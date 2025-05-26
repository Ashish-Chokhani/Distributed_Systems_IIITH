package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"
	"math/rand"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GeneralClient represents a client for a single general
type GeneralClient struct {
	ID        int32
	Role      pb.Role
	IsTraitor bool
	Client    pb.ByzantineServiceClient
}

func main() {
	// Parse command-line arguments
	serverAddr := flag.String("server", "localhost:50051", "The server address")
	numGenerals := flag.Int("n", 4, "Number of generals")
	numTraitors := flag.Int("t", 1, "Number of traitors")
	decision := flag.String("decision", "ATTACK", "Commander's initial decision (ATTACK/RETREAT)")
	commanderTraitor := flag.Bool("commander-traitor", false, "Set the commander as a traitor")
	flag.Parse()

	// Validate inputs
	if *numTraitors >= 4 {
		log.Fatalf("Number of traitors must be less than 4")
	}
	
	if int32(*numGenerals) <= int32(*numTraitors)*3 {
		log.Fatalf("For Byzantine consensus, n must be > 3t (got n=%d, t=%d)", *numGenerals, *numTraitors)
	}

	// Establish connection with server
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create gRPC client
	client := pb.NewByzantineServiceClient(conn)

	// Create generals
	var initialOrder pb.Decision
	if *decision == "ATTACK" {
		initialOrder = pb.Decision_ATTACK
	} else {
		initialOrder = pb.Decision_RETREAT
	}

	fmt.Printf("Starting Byzantine Generals simulation with %d generals (%d traitors)\n", 
		*numGenerals, *numTraitors)
	fmt.Printf("Commander's initial order: %s\n", initialOrder.String())
	if *commanderTraitor {
		fmt.Println("Commander is a TRAITOR in this simulation")
	} else {
		fmt.Println("Commander is LOYAL in this simulation")
	}

	// Register all generals with the system
	generals := make([]*GeneralClient, *numGenerals)
	
	// First general is always the commander
	commanders := make([]*GeneralClient, 0, 1)
	traitors := make([]*GeneralClient, 0, *numTraitors)
	
	// Commander registration with configurable loyalty
	commanderClient := &GeneralClient{
		ID:        0,
		Role:      pb.Role_COMMANDER,
		IsTraitor: *commanderTraitor, // Use the flag value instead of hardcoded false
		Client:    client,
	}
	
	// Register commander
	resp, err := client.RegisterGeneral(context.Background(), &pb.RegisterRequest{
		Id:        commanderClient.ID,
		Role:      commanderClient.Role,
		IsTraitor: commanderClient.IsTraitor,
	})
	if err != nil || !resp.Success {
		log.Fatalf("Failed to register commander: %v", err)
	}
	
	generals[0] = commanderClient
	commanders = append(commanders, commanderClient)
	
	rand.Seed(time.Now().UnixNano())

	// Determine actual number of traitors (at most t)
	actualTraitors := rand.Intn(*numTraitors + 1) // Random number between 0 and numTraitors
	fmt.Printf("Simulation will have %d traitors (maximum allowed: %d)\n", 
		actualTraitors, *numTraitors)

	// Then replace the current assignment logic
	remainingTraitors := actualTraitors
	if *commanderTraitor {
		traitors = append(traitors, commanderClient)
		remainingTraitors--
		fmt.Printf("General %d: Commander, Traitor\n", commanderClient.ID)
	} else {
		fmt.Printf("General %d: Commander, Loyal\n", commanderClient.ID)
	}

	lieutenantIndices := make([]int, *numGenerals-1)
	for i := range lieutenantIndices {
		lieutenantIndices[i] = i + 1  // +1 because indices start at 1 (commander is 0)
	}

	// Shuffle the indices
	rand.Shuffle(len(lieutenantIndices), func(i, j int) {
		lieutenantIndices[i], lieutenantIndices[j] = lieutenantIndices[j], lieutenantIndices[i]
	})

	// Use the first 'remainingTraitors' indices to assign traitors
	traitorMap := make(map[int]bool)
	for i := 0; i < remainingTraitors && i < len(lieutenantIndices); i++ {
		traitorMap[lieutenantIndices[i]] = true
	}

		
	// Register lieutenants
	for i := 1; i < *numGenerals; i++ {
		// Determine if this general is a traitor
		isTraitor := traitorMap[i]
		if isTraitor {
			remainingTraitors--
		}

		generalClient := &GeneralClient{
			ID:        int32(i),
			Role:      pb.Role_LIEUTENANT,
			IsTraitor: isTraitor,
			Client:    client,
		}
		
		// Register with the server
		resp, err := client.RegisterGeneral(context.Background(), &pb.RegisterRequest{
			Id:        generalClient.ID,
			Role:      generalClient.Role,
			IsTraitor: generalClient.IsTraitor,
		})
		if err != nil || !resp.Success {
			log.Fatalf("Failed to register lieutenant %d: %v", i, err)
		}

		generals[i] = generalClient
		
		if isTraitor {
			traitors = append(traitors, generalClient)
			fmt.Printf("General %d: Lieutenant, Traitor\n", generalClient.ID)
		} else {
			fmt.Printf("General %d: Lieutenant, Loyal\n", generalClient.ID)
		}
	}
	
	// Ensure correct number of traitors
	// Ensure number of traitors is at most the specified maximum
	if len(traitors) > *numTraitors {
		log.Fatalf("Too many traitors created: got %d, maximum allowed %d", 
			len(traitors), *numTraitors)
	}

	// Phase 1: Commander sends initial order to all lieutenants
	fmt.Println("\nPhase 1: Commander sends initial order to all lieutenants")
	commander := commanders[0]
	var wg sync.WaitGroup
	
	for i := 1; i < *numGenerals; i++ {
		wg.Add(1)
		go func(lieutenant *GeneralClient) {
			defer wg.Done()
			
			// Send order from commander to lieutenant
			resp, err := commander.Client.SendOrder(context.Background(), &pb.OrderRequest{
				SenderId:   commander.ID,
				ReceiverId: lieutenant.ID,
				Order:      initialOrder,
			})
			
			if err != nil {
				log.Printf("Error sending order to lieutenant %d: %v", lieutenant.ID, err)
				return
			}
			
			log.Printf("Commander sent order to Lieutenant %d: %s - Response: %s", 
				lieutenant.ID, initialOrder.String(), resp.Message)
		}(generals[i])
	}
	
	wg.Wait()
	
	// Phases 2 to t+1: Lieutenants exchange messages
	maxRounds := *numTraitors + 1
	
	for round := 2; round <= maxRounds; round++ {
		fmt.Printf("\nPhase %d: Exchanging messages between generals\n", round)
		var roundWg sync.WaitGroup
		
		// Each general forwards their information to all other generals
		for i := 0; i < *numGenerals; i++ {
			sender := generals[i]
			
			for j := 0; j < *numGenerals; j++ {
				if i == j {
					continue // Skip sending to self
				}
				
				receiver := generals[j]
				roundWg.Add(1)
				
				go func(s, r *GeneralClient, currentRound int) {
					defer roundWg.Done()
					
					// Value to forward should be based on the sender's understanding from previous round
					// In a real implementation, we would retrieve the actual value from previous round
					// Here we'll use a simplified approach:
					value := initialOrder // Default to initial order for loyal generals
					
					if s.IsTraitor {
						// Traitors might randomly send the opposite value
						if rand.Float32() < 0.5 {
							if initialOrder == pb.Decision_ATTACK {
								value = pb.Decision_RETREAT
							} else {
								value = pb.Decision_ATTACK
							}
						}
						// Otherwise, even traitors might tell the truth sometimes
					}
					
					// Send message
					_, err := s.Client.ExchangeMessage(context.Background(), &pb.MessageRequest{
						Round:      int32(currentRound),
						SenderId:   s.ID,
						ReceiverId: r.ID,
						Value:      value,
						Path:       []int32{s.ID},
					})
					
					if err != nil {
						log.Printf("Error in round %d from %d to %d: %v", 
							currentRound, s.ID, r.ID, err)
					}
				}(sender, receiver, round)
			}
		}
		
		roundWg.Wait()
		fmt.Printf("Round %d completed\n", round)
		
		// Small delay between rounds to allow server processing
		time.Sleep(time.Second)
	}

	// Final phase: Get consensus from each general
	fmt.Println("\nFinal Phase: Collecting decisions from all generals")
	time.Sleep(2 * time.Second) // Give server time to process all messages
	
	// // Create a table to display results
	// fmt.Println("\n=== FINAL DECISIONS ===")
	// fmt.Println("ID | Role       | Loyalty  | Decision")
	// fmt.Println("---+------------+----------+----------")
	
	// Track consensus
	var consensusReached bool
	var consensusDetails string
	
	for _, g := range generals {
		resp, err := g.Client.GetConsensus(context.Background(), &pb.ConsensusRequest{
			GeneralId: g.ID,
		})
		
		if err != nil {
			log.Printf("Error getting consensus from general %d: %v", g.ID, err)
			continue
		}
		
		// role := "Lieutenant"
		// if g.Role == pb.Role_COMMANDER {
		// 	role = "Commander"
		// }
		
		// loyalty := "Loyal"
		// if g.IsTraitor {
		// 	loyalty = "Traitor"
		// }
		
		// fmt.Printf("%2d | %-10s | %-8s | %s\n", 
		// 	g.ID, role, loyalty, resp.Decision.String())
		
		// Save consensus details from the first response (they should all be the same)
		if consensusDetails == "" {
			consensusReached = resp.ConsensusReached
			consensusDetails = resp.Details
		}
	}
	
	// Print final consensus information
	fmt.Println("\n=== CONSENSUS RESULTS ===")
	fmt.Println(consensusDetails)
	
	if consensusReached {
		fmt.Println("SIMULATION SUCCESSFUL: Byzantine consensus achieved!")
	} else {
		fmt.Println("SIMULATION FAILED: Byzantine consensus not achieved")
	}
}