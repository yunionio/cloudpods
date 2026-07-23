package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"yunion.io/x/ovsdb/client"
	"yunion.io/x/ovsdb/schema/ovn_nb"
)

func main() {
	// 1. Connect to OVSDB (Adjust address as needed)
	// Default OVN NB socket location
	target := "unix:/var/run/ovn/ovnnb_db.sock"
	
	fmt.Printf("Connecting to %s...\n", target)
	cli, err := client.NewClient(target)
	if err != nil {
		log.Fatalf("Failed to connect: %v\n(Make sure OVN is running and the socket path is correct)", err)
	}
	defer cli.Close()
	log.Println("Connected to OVSDB")

	// 2. Setup Monitoring
	// We use the generated OVNNorthbound struct as the target cache
	dbCache := &ovn_nb.OVNNorthbound{}
	ctx := context.Background()

	// Start monitoring in a goroutine (it blocks waiting for updates)
	go func() {
		log.Println("Starting Monitor...")
		if err := cli.MonitorDB(ctx, "OVN_Northbound", dbCache); err != nil {
			log.Printf("Monitor failed: %v", err)
		}
	}()

	// Wait for initial sync
	fmt.Println("Waiting for cache sync...")
	time.Sleep(1 * time.Second)
	fmt.Printf("Cache Synced: %d Logical Switches found\n", len(dbCache.LogicalSwitch))

	// 3. Perform a Transaction (Insert)
	lsName := fmt.Sprintf("test-ls-%d", time.Now().Unix())
	newLs := &ovn_nb.LogicalSwitch{
		Name: lsName,
		ExternalIds: map[string]string{
			"created-by": "go-client",
		},
	}

	log.Printf("Creating Logical Switch: %s", lsName)
	op := client.OvsdbCreateOp(newLs, "newSwitch")
	
	// Execute Transaction
	results, err := cli.TransactOps(ctx, "OVN_Northbound", op)
	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}

	// Check Result
	if len(results) > 0 && results[0].Error == "" {
		fmt.Printf("Transaction Success! UUID: %v\n", results[0].Uuid)
	} else {
		fmt.Printf("Transaction Response: %+v\n", results)
	}

	// 4. Verify Update in Cache
	time.Sleep(200 * time.Millisecond) // Allow monitor update to arrive
	found := false
	for _, ls := range dbCache.LogicalSwitch {
		if ls.Name == lsName {
			fmt.Printf("Verified: Switch %s exists in local cache (UUID: %s)\n", lsName, ls.Uuid)
			found = true
			
			// Cleanup: Delete the test switch
			log.Printf("Cleaning up (Deleting %s)...", lsName)
			delOp := client.OvsdbDeleteOp("Logical_Switch", client.NewConditionUuid(ls.Uuid))
			_, err := cli.TransactOps(ctx, "OVN_Northbound", delOp)
			if err != nil {
				log.Printf("Cleanup failed: %v", err)
			} else {
				log.Println("Cleanup successful.")
			}
			break
		}
	}

	if !found {
		log.Printf("Error: Created switch not found in cache yet.")
	}
}
