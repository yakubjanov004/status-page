package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ScanSystemd(dir string) ([]DiscoveredItem, error) {
	if dir == "" {
		dir = "/etc/systemd/system"
	}
	var items []DiscoveredItem

	os.MkdirAll(dir, 0755)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".service") {
			continue
		}

		serviceName := file.Name()
		cleanName := strings.TrimSuffix(serviceName, ".service")
		
		projectName, componentType := parseSystemdName(cleanName)

		url := ""
		path := filepath.Join(dir, file.Name())
		if f, err := os.Open(path); err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "ExecStart=") {
					// Look for --port \d+ or -p \d+ or -l \d+
					portRegex := regexp.MustCompile(`(?:--port|-p|-l)\s+(\d+)`)
					if match := portRegex.FindStringSubmatch(line); match != nil {
						url = "http://127.0.0.1:" + match[1]
					}
					// Also look for uvicorn host/port combination without flags if defined like uvicorn main:app --host 127.0.0.1 --port 8000
					// Wait, the regex above catches --port \d+, which is good enough for uvicorn and serve
					break
				}
			}
			f.Close()
		}

		items = append(items, DiscoveredItem{
			ID:        "systemd:" + cleanName,
			Source:    "systemd",
			Project:   projectName,
			Component: componentType,
			URL:       url,
			RawName:   serviceName,
		})
	}

	return items, nil
}

func parseSystemdName(name string) (string, string) {
	parts := strings.Split(name, "-")
	if len(parts) == 1 {
		return name, "unknown"
	}

	// Examples: tokpoint-backend, tokpoint-frontend, alfaconnect-bot
	projectName := parts[0]
	componentType := "unknown"
	
	lastPart := parts[len(parts)-1]
	if lastPart == "backend" || lastPart == "api" || lastPart == "bot" || lastPart == "worker" {
		componentType = "backend"
	} else if lastPart == "frontend" || lastPart == "webapp" {
		componentType = "frontend"
	}

	return projectName, componentType
}
