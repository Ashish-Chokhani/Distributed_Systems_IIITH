package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// General represents a node in the Byzantine system
type General struct {
	ID            int32
	Role          pb.Role
	IsTraitor     bool
	Messages      map[int32]map[int32][]pb.Decision // round -> sender -> values
	FinalDecision pb.Decision
	mutex         sync.Mutex
}

// ByzantineServer implements the gRPC server
type ByzantineServer struct {
	pb.UnimplementedByzantineServiceServer
	Generals       map[int32]*General
	NumGenerals    int32
	NumTraitors    int32
	MaxRounds      int32
	CurrentRound   int32
	mutex          sync.RWMutex
	roundCompleted map[int32]bool
}

// NewByzantineServer creates a new server instance
func NewByzantineServer() *ByzantineServer {
	return &ByzantineServer{
		Generals:       make(map[int32]*General),
		CurrentRound:   0,
		roundCompleted: make(map[int32]bool),
	}
}

// RegisterGeneral registers a new general with the system
func (s *ByzantineServer) RegisterGeneral(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.Generals[req.Id]; exists {
		return &pb.RegisterResponse{
			Success: false,
			Message: fmt.Sprintf("General with ID %d already exists", req.Id),
		}, nil
	}

	s.Generals[req.Id] = &General{
		ID:        req.Id,
		Role:      req.Role,
		IsTraitor: req.IsTraitor,
		Messages:  make(map[int32]map[int32][]pb.Decision),
		mutex:     sync.Mutex{},
	}

	if req.IsTraitor {
		s.NumTraitors++
	}

	s.NumGenerals++
	s.MaxRounds = s.NumTraitors + 1

	log.Printf("Registered General ID: %d, Role: %s, IsTraitor: %v", 
		req.Id, req.Role.String(), req.IsTraitor)

	return &pb.RegisterResponse{
		Success:       true,
		Message:       fmt.Sprintf("General %d registered successfully", req.Id),
		TotalGenerals: s.NumGenerals,
	}, nil
}

// SendOrder handles the initial order from the commander
func (s *ByzantineServer) SendOrder(ctx context.Context, req *pb.OrderRequest) (*pb.OrderResponse, error) {
    s.mutex.RLock()
    general, exists := s.Generals[req.SenderId]
    receiverGeneral, receiverExists := s.Generals[req.ReceiverId]
    s.mutex.RUnlock()

    if !exists {
        return &pb.OrderResponse{
            Received: false,
            Message:  fmt.Sprintf("Sender General %d does not exist", req.SenderId),
        }, nil
    }

    if !receiverExists {
        return &pb.OrderResponse{
            Received: false,
            Message:  fmt.Sprintf("Receiver General %d does not exist", req.ReceiverId),
        }, nil
    }

    if general.Role != pb.Role_COMMANDER {
        return &pb.OrderResponse{
            Received: false,
            Message:  "Only commander can send initial orders",
        }, nil
    }

    // Store the order in the commander's own message history as well
    // This ensures the commander remembers its own original order
    general.mutex.Lock()
    if general.Messages[1] == nil {
        general.Messages[1] = make(map[int32][]pb.Decision)
    }
    // Commander stores its own order (to itself)
    general.Messages[1][req.SenderId] = append(
        general.Messages[1][req.SenderId], req.Order)
    general.mutex.Unlock()

    // Store the message in receiver's message log (round 1)
    receiverGeneral.mutex.Lock()
    if receiverGeneral.Messages[1] == nil {
        receiverGeneral.Messages[1] = make(map[int32][]pb.Decision)
    }

    // If the general is a traitor, they may alter the message
    value := req.Order
    if general.IsTraitor && rand.Float32() < 0.5 {
        if value == pb.Decision_ATTACK {
            value = pb.Decision_RETREAT
        } else {
            value = pb.Decision_ATTACK
        }
    }

    receiverGeneral.Messages[1][req.SenderId] = append(
        receiverGeneral.Messages[1][req.SenderId], value)
    receiverGeneral.mutex.Unlock()

    log.Printf("Order sent from General %d to General %d: %s", 
        req.SenderId, req.ReceiverId, value.String())

    // Mark round 1 as started if not already
    s.mutex.Lock()
    if s.CurrentRound == 0 {
        s.CurrentRound = 1
        log.Printf("Round 1 started: Commander sending orders")
    }
    s.mutex.Unlock()

    return &pb.OrderResponse{
        Received: true,
        Message:  "Order received",
    }, nil
}

// ExchangeMessage handles message exchange between generals in later rounds
func (s *ByzantineServer) ExchangeMessage(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResponse, error) {
	s.mutex.RLock()
	sender, senderExists := s.Generals[req.SenderId]
	receiver, receiverExists := s.Generals[req.ReceiverId]
	currentRound := s.CurrentRound
	s.mutex.RUnlock()

	if !senderExists || !receiverExists {
		return &pb.MessageResponse{
			Received: false,
			Message:  "Sender or receiver does not exist",
		}, nil
	}

	if req.Round < currentRound {
		return &pb.MessageResponse{
			Received: false,
			Message:  fmt.Sprintf("Message for previous round %d rejected, current round is %d", req.Round, currentRound),
		}, nil
	}

	// Start a new round if necessary
	if req.Round > currentRound {
		s.mutex.Lock()
		if req.Round > s.CurrentRound {
			s.CurrentRound = req.Round
			log.Printf("Round %d started", req.Round)
		}
		s.mutex.Unlock()
	}

	// Check if message is going in a cycle
	for _, id := range req.Path {
		if id == req.ReceiverId {
			return &pb.MessageResponse{
				Received: false,
				Message:  "Message would create a cycle, rejected",
			}, nil
		}
	}

	// Modify message if sender is traitor
	value := req.Value
	if sender.IsTraitor && rand.Float32() < 0.5 {
		if value == pb.Decision_ATTACK {
			value = pb.Decision_RETREAT
		} else {
			value = pb.Decision_ATTACK
		}
		log.Printf("Traitor General %d changed message to General %d from %s to %s", 
			req.SenderId, req.ReceiverId, req.Value.String(), value.String())
	}

	// Store the message
	receiver.mutex.Lock()
	if receiver.Messages[req.Round] == nil {
		receiver.Messages[req.Round] = make(map[int32][]pb.Decision)
	}
	receiver.Messages[req.Round][req.SenderId] = append(
		receiver.Messages[req.Round][req.SenderId], value)
	receiver.mutex.Unlock()

	log.Printf("Message received - Round: %d, From: %d, To: %d, Value: %s", 
		req.Round, req.SenderId, req.ReceiverId, value.String())

	return &pb.MessageResponse{
		Received: true,
		Message:  "Message received",
	}, nil
}

// GetConsensus returns the final decision for a general
func (s *ByzantineServer) GetConsensus(ctx context.Context, req *pb.ConsensusRequest) (*pb.ConsensusResponse, error) {
	s.mutex.RLock()
	general, exists := s.Generals[req.GeneralId]
	maxRounds := s.MaxRounds
	s.mutex.RUnlock()

	if !exists {
		return &pb.ConsensusResponse{
			Decision:        pb.Decision_RETREAT,
			ConsensusReached: false,
			Details:         fmt.Sprintf("General %d does not exist", req.GeneralId),
		}, nil
	}

	// Calculate the decision
	decision := s.makeDecision(general, maxRounds)

	// Calculate overall consensus
	attackCount := 0
	retreatCount := 0
	loyalAttackCount := 0
	loyalRetreatCount := 0

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, g := range s.Generals {
		gDecision := s.makeDecision(g, maxRounds)
		if gDecision == pb.Decision_ATTACK {
			attackCount++
			if !g.IsTraitor {
				loyalAttackCount++
			}
		} else {
			retreatCount++
			if !g.IsTraitor {
				loyalRetreatCount++
			}
		}
	}

	// Determine if consensus was reached among loyal generals
	var consensusDecision pb.Decision
	if loyalAttackCount > loyalRetreatCount {
		consensusDecision = pb.Decision_ATTACK
	} else {
		consensusDecision = pb.Decision_RETREAT
	}

	allLoyalAgree := true
	for _, g := range s.Generals {
		if !g.IsTraitor && s.makeDecision(g, maxRounds) != consensusDecision {
			allLoyalAgree = false
			break
		}
	}

	// Create vote count map for the response
	voteCounts := make(map[string]int32)
	voteCounts["ATTACK"] = int32(attackCount)
	voteCounts["RETREAT"] = int32(retreatCount)
	voteCounts["LOYAL_ATTACK"] = int32(loyalAttackCount)
	voteCounts["LOYAL_RETREAT"] = int32(loyalRetreatCount)

	details := fmt.Sprintf("General %d final decision: %s\n", 
		req.GeneralId, decision.String())
	details += fmt.Sprintf("Total votes - Attack: %d, Retreat: %d\n", 
		attackCount, retreatCount)
	details += fmt.Sprintf("Loyal votes - Attack: %d, Retreat: %d\n", 
		loyalAttackCount, loyalRetreatCount)
	
	if allLoyalAgree {
		details += fmt.Sprintf("CONSENSUS REACHED: All loyal generals agreed on: %s\n", 
			consensusDecision.String())
	} else {
		details += "CONSENSUS FAILED: Loyal generals did not reach agreement\n"
	}

	return &pb.ConsensusResponse{
		Decision:         decision,
		ConsensusReached: allLoyalAgree,
		Details:          details,
		VoteCounts:       voteCounts,
	}, nil
}

// Modified makeDecision function to correctly handle decision making
func (s *ByzantineServer) makeDecision(general *General, maxRounds int32) pb.Decision {
    if general.IsTraitor {
        // Traitors make arbitrary decisions
        if rand.Float32() < 0.5 {
            return pb.Decision_ATTACK
        }
        return pb.Decision_RETREAT
    }

    // Loyal generals follow the algorithm
    countAttack := 0
    countRetreat := 0

    general.mutex.Lock()
    defer general.mutex.Unlock()
    
    // For the Commander, we need to consider the original order
    // If the general is the commander, they should follow their own initial order
    if general.Role == pb.Role_COMMANDER {
        // Find the Commander's initial order from round 1
        for round := int32(1); round <= maxRounds; round++ {
            // Check if there are any messages in this round
            if messages, ok := general.Messages[round]; ok {
                for _, decisions := range messages {
                    // Process all messages from all rounds
                    for _, decision := range decisions {
                        if decision == pb.Decision_ATTACK {
                            countAttack++
                        } else {
                            countRetreat++
                        }
                    }
                }
            }
        }
        
        // If the Commander didn't receive any messages (which would be unusual),
        // default to the initial order that was sent to lieutenants
        if countAttack == 0 && countRetreat == 0 {
            // Since we don't have direct access to the initial order here,
            // we'll need to inspect the lieutenants' messages from round 1
            // to infer what the commander sent
            s.mutex.RLock()
            defer s.mutex.RUnlock()
            
            for id, g := range s.Generals {
                if id != general.ID && g.Role == pb.Role_LIEUTENANT {
                    if g.Messages[1] != nil && len(g.Messages[1][general.ID]) > 0 {
                        initialDecision := g.Messages[1][general.ID][0]
                        if initialDecision == pb.Decision_ATTACK {
                            return pb.Decision_ATTACK
                        } else {
                            return pb.Decision_RETREAT
                        }
                    }
                }
            }
        }
    } else {
        // For lieutenants, consider messages from all rounds, not just the final round
        for round := int32(1); round <= maxRounds; round++ {
            if messages, ok := general.Messages[round]; ok {
                for _, decisions := range messages {
                    for _, decision := range decisions {
                        if decision == pb.Decision_ATTACK {
                            countAttack++
                        } else {
                            countRetreat++
                        }
                    }
                }
            }
        }
    }

    // Make the decision based on the majority
    if countAttack > countRetreat {
        return pb.Decision_ATTACK
    } else if countRetreat > countAttack {
        return pb.Decision_RETREAT
    } else {
        // In case of a tie, we need a deterministic rule
        // Conventionally in Byzantine algorithms, RETREAT is safer
        return pb.Decision_RETREAT
    }
}

func main() {
	port := flag.Int("port", 50051, "The server port")
	flag.Parse()

	// Create a new gRPC server
	server := grpc.NewServer()
	
	// Create and register the Byzantine service
	byzantineServer := NewByzantineServer()
	pb.RegisterByzantineServiceServer(server, byzantineServer)
	
	// Register reflection service on gRPC server for debugging
	reflection.Register(server)

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	log.Printf("Byzantine Generals server listening on port %d", *port)
	
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}