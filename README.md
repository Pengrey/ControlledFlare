# Controlled Flare
This program helps users assess the risk of their org by checking if cloudlfared host are reachable by their original IPs or hosts

## Usage
First, you need to export your Cloudflare API token:

```bash
export CLOUDFLARE_API_TOKEN=<REDACTED>
```

Build and run the program:
```bash
$ go build src/main.go
$ ./main
...
$
```


### Tips
Some tips for the results

#### Only Ips
If you only want the results where the registers have only an Ipv4 or an Ipv6 use the following
```bash
grep -E '((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)|([a-f0-9]{1,4}:){7}[a-f0-9]{1,4}' results.csv
```