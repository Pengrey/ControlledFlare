package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"strconv"

	"golang.org/x/exp/slices"

	"github.com/tidwall/gjson"
	"github.com/schollz/progressbar/v3"
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

	// Fetch Number of Zones 
	zone, err := client.Zones.List(context.Background(), zones.ZoneListParams{})
	if err != nil {
		panic(err.Error())
	}

	// scuffed usage of undocumented functions cuz cloudflare doesnt provide a better way 
	// (https://github.com/cloudflare/cloudflare-go/issues/4161)
	number_of_zone, err := strconv.ParseInt(gjson.Get(zone.ResultInfo.JSON.RawJSON(), "total_count").String(), 10, 64)
	if err != nil {
		panic(err.Error())
	}

	// Progress bar
	bar := progressbar.Default(number_of_zone)

	// Fetch Zones
	zone_iter := client.Zones.ListAutoPaging(context.Background(), zones.ZoneListParams{})

	// Automatically fetches more pages as needed.
	for zone_iter.Next() {
		// Get zone
		zone := zone_iter.Current()

		// Set to store the unique IPs
		ips := []string{}
		
		// Fetch the A, AAAA and CNAME records
		recordTypes := []dns.RecordListParamsType{dns.RecordListParamsTypeA, dns.RecordListParamsTypeAAAA, dns.RecordListParamsTypeCNAME}

		for _, recordType := range recordTypes {
			records, err := client.DNS.Records.List(context.Background(), dns.RecordListParams{
				ZoneID: cloudflare.F(zone.ID),
				Type: cloudflare.F(recordType),
				Proxied: cloudflare.Bool(true),
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

		// For each Ip in the set
		for _, ip := range ips {
			// Increment the WaitGroup counter
			wg.Add(1)

			// Go routines goes brrrr
			go pingHost(zone.Name, ip, &wg, arq)
		}

		// Update progress bar
		bar.Add(1)
	}
	if err := zone_iter.Err(); err != nil {
		panic(err.Error())
	}

	// Wait for all ICMP fetches to complete.
	wg.Wait()

	// Close file handle when needed
	defer arq.Close()
}