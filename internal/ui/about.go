package ui

import (
	"fmt"

	"swiftinstall/internal/appinfo"
	"swiftinstall/internal/i18n"
)

func GetAboutText() string {
	var content string

	content += TitleStyle.Render("SwiftInstall") + "\n\n"

	items := []struct {
		label string
		value string
	}{
		{i18n.T("about_author"), appinfo.Author},
		{i18n.T("about_contact"), appinfo.Contact},
		{"GitHub", appinfo.GitHubURL},
	}

	for _, item := range items {
		label := KeyStyle.Render(item.label + ":")
		value := HelpStyle.Render(item.value)
		content += fmt.Sprintf("  %s %s\n", label, value)
	}

	content += "\n" + HelpStyle.Render(appinfo.Copyright)

	return content
}

func RunAbout() {
	fmt.Println(GetCompactLogo())
	fmt.Println()
	fmt.Println(GetAboutText())
}
