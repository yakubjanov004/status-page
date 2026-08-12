package discovery

import (
	"os"
	"strings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func DiscoverAll() []DiscoveredItem {
	nginxDir := os.Getenv("DISCOVERY_NGINX_DIR")
	systemdDir := os.Getenv("DISCOVERY_SYSTEMD_DIR")

	nginxItems, _ := ScanNginx(nginxDir)
	systemdItems, _ := ScanSystemd(systemdDir)

	var allItems []DiscoveredItem
	allItems = append(allItems, nginxItems...)
	allItems = append(allItems, systemdItems...)

	caser := cases.Title(language.English)
	
	// Normalize project names
	for i := range allItems {
		projName := allItems[i].Project
		projName = strings.ReplaceAll(projName, "-", " ")
		projName = strings.ReplaceAll(projName, "_", " ")
		allItems[i].Project = caser.String(projName)
	}

	return allItems
}
