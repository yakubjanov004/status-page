package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DiscoveredItem struct {
	ID        string `json:"id"`
	Source    string `json:"source"`    // "nginx" or "systemd"
	Project   string `json:"project"`   // Best guess project name
	Component string `json:"component"` // "frontend" or "backend" or "unknown"
	URL       string `json:"url"`
	RawName   string `json:"raw_name"`
}

func ScanNginx(dir string) ([]DiscoveredItem, error) {
	if dir == "" {
		dir = "/etc/nginx/sites-enabled"
	}
	var items []DiscoveredItem

	os.MkdirAll(dir, 0755)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	serverNameRegex := regexp.MustCompile(`server_name\s+([^;]+);`)
	locationRegex := regexp.MustCompile(`location\s+([^{]+)\s*\{`)
	proxyPassRegex := regexp.MustCompile(`proxy_pass\s+([^;]+);`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(dir, file.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()

		var currentServers []string
		currentLocation := ""
		depth := 0

		var frontendURL string
		var backendURL string

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Handle block depth
			if strings.Contains(line, "{") {
				depth++
			}
			if strings.Contains(line, "}") {
				depth--
				if depth <= 1 {
					currentLocation = ""
				}
				if depth == 0 {
					// End of server block, process what we found
					for _, serverName := range currentServers {
						projectName := extractProjectNameFromDomain(serverName)
						
						fURL := frontendURL
						if fURL == "" {
							fURL = "http://" + serverName
						}
						
						items = append(items, DiscoveredItem{
							ID:        "nginx:frontend:" + serverName,
							Source:    "nginx",
							Project:   projectName,
							Component: "frontend",
							URL:       fURL,
							RawName:   serverName + " (Location /)",
						})

						if backendURL != "" {
							items = append(items, DiscoveredItem{
								ID:        "nginx:backend:" + serverName,
								Source:    "nginx",
								Project:   projectName,
								Component: "backend",
								URL:       backendURL,
								RawName:   serverName + " (Location /api)",
							})
						}
					}
					// Reset for next server block
					currentServers = nil
					frontendURL = ""
					backendURL = ""
				}
				continue
			}

			if snMatch := serverNameRegex.FindStringSubmatch(line); snMatch != nil {
				names := strings.Fields(snMatch[1])
				currentServers = append(currentServers, names...)
			}

			if locMatch := locationRegex.FindStringSubmatch(line); locMatch != nil {
				currentLocation = strings.TrimSpace(locMatch[1])
			}

			if ppMatch := proxyPassRegex.FindStringSubmatch(line); ppMatch != nil {
				target := strings.TrimSpace(ppMatch[1])
				
				if currentLocation == "/" {
					frontendURL = target
				} else if strings.HasPrefix(currentLocation, "/api") || strings.HasPrefix(currentLocation, "^~ /api") {
					backendURL = target
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	return items, nil
}

func extractProjectNameFromDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		if parts[0] == "api" {
			return parts[1]
		}
		// e.g. tokpointapi.darrov.uz -> tokpointapi (we can further strip 'api' if we want)
		name := parts[0]
		if strings.HasSuffix(name, "api") || strings.HasSuffix(name, "bot") {
			name = strings.ReplaceAll(name, "api", "")
			name = strings.ReplaceAll(name, "bot", "")
		}
		return name
	}
	return domain
}
