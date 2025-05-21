package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/cloudflare/cloudflare-go/v4/dns"
)

// Function to ping the hosts
func pingHost(host string, ip string, wg *sync.WaitGroup, arq *os.File) {
	// Sussy command injection through DNS registrar B)
	full_command := fmt.Sprintf("ping -c 1 -w 1 %s | grep '64 bytes'", ip)
	err := exec.Command("/bin/bash", "-c", full_command).Run()

	// If it doesnt give an error then its reacheable
	if err == nil {
		message := fmt.Sprintf("%s,%s,%s,\n", host, ip, "True")
		data := []byte(message)
		arq.Write(data)
	} else {
		message := fmt.Sprintf("%s,%s,%s,\n", host, ip, "False")
		data := []byte(message)
		arq.Write(data)
	}

	// Decrement the counter when the goroutine completes.
	defer wg.Done()
}

func main() {
	// Wait group for goroutines
	var wg sync.WaitGroup

	// Create results file
	arq, err := os.Create("results.csv")
	if err != nil {
		panic(err.Error())
	}

	// Write header of CSV file
	arq.Write([]byte(fmt.Sprintf("%s,%s,%s,\n", "HOST", "IP", "REACHEABLE")))

	// Create API Client
	client := cloudflare.NewClient(
		option.WithAPIKey(os.Getenv("CLOUDFLARE_API_TOKEN")),
	)

	// Fetch Zones
	zone_iter := client.Zones.ListAutoPaging(context.Background(), zones.ZoneListParams{})

	// Automatically fetches more pages as needed.
	for zone_iter.Next() {
		zone := zone_iter.Current()
		// Print zone id
		fmt.Printf("Name: %s", zone.Name)

		// Set to store the unique IPs
		ips := []string{}
		
		// Fetch the A, AAAA and CNAME records
		recordTypes := []dns.RecordListParamsType{dns.RecordListParamsTypeA, dns.RecordListParamsTypeAAAA, dns.RecordListParamsTypeCNAME}

		for _, recordType := range recordTypes {
			records, err := client.DNS.Records.List(context.Background(), dns.RecordListParams{
				ZoneID: cloudflare.F(zone.ID),
				Type: cloudflare.F(recordType),
			})
			if err != nil {
				panic(err.Error())
			}

			// Iterate record results
			for _, record := range records.Result {
				// Add IP if not already added
				if ! slices.Contains(ips, record.Content) {
					ips = append(ips, record.Content)
				}
			}
		}

		// Print resulting set
		fmt.Printf(", IPs: %v\n", ips)

		// For each Ip in the set
		for _, ip := range ips {
			// Increment the WaitGroup counter
			wg.Add(1)

			// Go routines goes brrrr
			go pingHost(zone.Name, ip, &wg, arq)
		}
	}
	if err := zone_iter.Err(); err != nil {
		panic(err.Error())
	}

	// Wait for all ICMP fetches to complete.
	wg.Wait()

	// Close file handle when needed
	defer arq.Close()
}