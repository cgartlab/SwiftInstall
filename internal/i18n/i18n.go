package i18n

import (
	"sync"
)

var (
	currentLang = "zh"
	mu          sync.RWMutex

	translations = map[string]map[string]string{
		"zh": {
			// App
			"app_name":       "SwiftInstall",
			"app_short_desc": "快速、简单、可靠的跨平台软件安装工具",
			"app_long_desc":  "SwiftInstall 是一个跨平台软件安装工具，支持 Windows (Winget) 和 macOS (Homebrew)",

			// Commands
			"cmd_version_short":   "显示版本信息",
			"cmd_install_short":   "安装软件",
			"cmd_install_long":    "从配置文件或命令行参数安装软件",
			"cmd_uninstall_short": "卸载软件",
			"cmd_uninstall_long":  "从配置文件或命令行参数卸载软件",
			"cmd_search_short":    "搜索软件",
			"cmd_search_long":     "使用包管理器搜索软件",
			"cmd_list_short":      "列出已配置的软件",
			"cmd_list_long":       "显示配置文件中所有已配置的软件",
			"cmd_config_short":    "管理配置",
			"cmd_config_long":     "交互式管理配置文件",
			"cmd_wizard_short":    "启动向导",
			"cmd_wizard_long":     "启动交互式安装向导",
			"cmd_batch_short":     "批量安装",
			"cmd_batch_long":      "从文件或配置批量安装软件",
			"cmd_export_short":    "导出配置",
			"cmd_export_long":     "导出配置为脚本或 JSON",
			"cmd_update_short":    "检查更新",
			"cmd_update_long":     "检查 SwiftInstall 更新",
			"cmd_clean_short":     "清理缓存",
			"cmd_clean_long":      "清理安装缓存和临时文件",
			"cmd_status_short":    "系统状态",
			"cmd_status_long":     "显示系统状态和已安装软件",
			"cmd_about_short":     "关于软件与作者",
			"cmd_about_long":      "显示作者信息、联系方式与项目地址",
			"cmd_help_short":      "查看完整帮助文档",
			"cmd_help_long":       "显示所有命令、参数、快捷键与示例",

			// Flags
			"flag_config":        "配置文件路径",
			"flag_language":      "设置语言 (en/zh)",
			"flag_export_format": "导出格式 (json/yaml/powershell/bash)",
			"flag_export_output": "输出文件路径",
			"flag_parallel":      "并行安装",

			// Warnings
			"warn_no_packages": "警告: 配置中没有软件包",

			// UI - Main Menu
			"menu_title":     "主菜单",
			"menu_install":   "安装软件",
			"menu_uninstall": "卸载软件",
			"menu_search":    "搜索软件",
			"menu_config":    "配置管理",
			"menu_wizard":    "安装向导",
			"menu_status":    "系统状态",
			"menu_clean":     "清理缓存",
			"menu_update":    "检查更新",
			"menu_exit":      "退出",
			"menu_about":     "关于作者",

			"menu_exit_desc":  "关闭程序并返回终端",
			"menu_about_desc": "查看作者与项目信息",

			// UI - Common
			"common_back":         "返回",
			"common_cancel":       "取消",
			"common_confirm":      "确认",
			"common_yes":          "是",
			"common_no":           "否",
			"common_success":      "成功",
			"common_failed":       "失败",
			"common_pending":      "等待中",
			"common_installing":   "安装中",
			"common_uninstalling": "卸载中",
			"common_searching":    "搜索中",
			"common_done":         "完成",
			"common_error":        "错误",
			"common_warning":      "警告",

			// Config errors
			"config_save_error": "保存配置失败",
			"common_info":       "信息",
			"common_navigation": "导航",
			"common_up":         "上",
			"common_down":       "下",
			"common_select":     "选择",
			"common_quit":       "退出",
			"common_tip":        "提示",
			"common_status":     "状态",

			// UI - Install
			"install_title":     "软件安装",
			"install_progress":  "安装进度",
			"install_total":     "总计",
			"install_completed": "已完成",
			"install_failed":    "失败",
			"install_skipped":   "跳过",
			"install_select":    "选择要安装的软件",
			"install_confirm":   "确认安装以下软件?",

			// UI - Search
			"search_title":       "搜索软件",
			"search_placeholder": "输入软件名称搜索...",
			"search_results":     "搜索结果",
			"search_no_results":  "未找到结果",
			"search_add":         "添加到列表",

			"search_refine":      "修改搜索",
			"search_added_hint":  "已添加当前软件到配置",
			"search_added_count": "已添加到配置",

			// UI - Config
			"config_title":    "配置管理",
			"config_add":      "添加软件",
			"config_remove":   "移除软件",
			"config_edit":     "编辑软件",
			"config_save":     "保存配置",
			"config_load":     "加载配置",
			"config_name":     "名称",
			"config_id":       "ID/包名",
			"config_category": "分类",

			"config_required":          "名称和 ID 为必填项",
			"config_added":             "已添加",
			"config_updated":           "已更新",
			"config_removed":           "已移除",
			"config_remove_confirm":    "确认移除以下软件？",
			"config_invalid_selection": "无效的选择，无法更新",

			// UI - Status
			"status_title":       "系统状态",
			"status_platform":    "平台",
			"status_package_mgr": "包管理器",
			"status_installed":   "已安装",
			"status_available":   "可用",
			"status_unavailable": "不可用",
			"status_version":     "版本",
			"status_total":       "总计",
			"status_more":        "还有",
			"status_packages":    "个软件包",
			"status_config":      "配置",
			"status_config_path": "配置文件路径",
			"status_configured":  "已配置软件",
			"status_install_pm":  "请先安装包管理器",

			// UI - Wizard
			"wizard_welcome":           "欢迎使用 SwiftInstall",
			"wizard_desc":              "本向导将帮助您快速安装常用软件",
			"wizard_step":              "步骤",
			"wizard_finish":            "完成安装",
			"wizard_select_categories": "选择要安装的软件类别:",
			"wizard_confirm":           "确认以下选择:",

			// UI - Update
			"update_current":      "当前版本",
			"update_latest":       "最新版本",
			"update_checking":     "正在检查更新...",
			"update_failed":       "检查更新失败",
			"update_parse_failed": "解析更新信息失败",
			"update_uptodate":     "已是最新版本!",
			"update_available":    "发现新版本!",
			"update_download":     "下载地址",
			"update_manual":       "请手动检查更新",
			"update_hint":         "下载新版本后替换当前可执行文件即可完成更新",

			// UI - Clean
			"clean_scanning":  "正在扫描缓存...",
			"clean_confirm":   "是否清理以上缓存? [y/N]: ",
			"clean_cleaning":  "正在清理...",
			"clean_no_cache":  "没有发现需要清理的缓存",
			"clean_cancelled": "已取消清理",
			"clean_partial":   "部分缓存清理失败:",
			"clean_done":      "缓存清理完成!",

			"about_title":   "关于 SwiftInstall",
			"about_author":  "作者",
			"about_contact": "联系方式",
		},
		"en": {
			// App
			"app_name":       "SwiftInstall",
			"app_short_desc": "Fast, simple, reliable cross-platform software installer",
			"app_long_desc":  "SwiftInstall is a cross-platform software installer supporting Windows (Winget) and macOS (Homebrew)",

			// Commands
			"cmd_version_short":   "Show version information",
			"cmd_install_short":   "Install software",
			"cmd_install_long":    "Install software from config file or command line arguments",
			"cmd_uninstall_short": "Uninstall software",
			"cmd_uninstall_long":  "Uninstall software from config file or command line arguments",
			"cmd_search_short":    "Search for software",
			"cmd_search_long":     "Search for software using package manager",
			"cmd_list_short":      "List configured software",
			"cmd_list_long":       "Show all configured software in config file",
			"cmd_config_short":    "Manage configuration",
			"cmd_config_long":     "Interactive configuration management",
			"cmd_wizard_short":    "Launch wizard",
			"cmd_wizard_long":     "Launch interactive installation wizard",
			"cmd_batch_short":     "Batch install",
			"cmd_batch_long":      "Batch install software from file or config",
			"cmd_export_short":    "Export configuration",
			"cmd_export_long":     "Export configuration as script or JSON",
			"cmd_update_short":    "Check for updates",
			"cmd_update_long":     "Check for SwiftInstall updates",
			"cmd_clean_short":     "Clean cache",
			"cmd_clean_long":      "Clean installation cache and temporary files",
			"cmd_status_short":    "System status",
			"cmd_status_long":     "Show system status and installed software",
			"cmd_about_short":     "About software and author",
			"cmd_about_long":      "Show author info, contacts, and project links",
			"cmd_help_short":      "Show complete help document",
			"cmd_help_long":       "Show all modules, flags, shortcuts, and examples",

			// Flags
			"flag_config":        "Config file path",
			"flag_language":      "Set language (en/zh)",
			"flag_export_format": "Export format (json/yaml/powershell/bash)",
			"flag_export_output": "Output file path",
			"flag_parallel":      "Parallel installation",

			// Warnings
			"warn_no_packages": "Warning: No packages in configuration",

			// UI - Main Menu
			"menu_title":     "Main Menu",
			"menu_install":   "Install Software",
			"menu_uninstall": "Uninstall Software",
			"menu_search":    "Search Software",
			"menu_config":    "Configuration",
			"menu_wizard":    "Installation Wizard",
			"menu_status":    "System Status",
			"menu_clean":     "Clean Cache",
			"menu_update":    "Check Updates",
			"menu_exit":      "Exit",
			"menu_about":     "About Author",

			"menu_exit_desc":  "Close the application and return to terminal",
			"menu_about_desc": "View author and project information",

			// UI - Common
			"common_back":         "Back",
			"common_cancel":       "Cancel",
			"common_confirm":      "Confirm",
			"common_yes":          "Yes",
			"common_no":           "No",
			"common_success":      "Success",
			"common_failed":       "Failed",
			"common_pending":      "Pending",
			"common_installing":   "Installing",
			"common_uninstalling": "Uninstalling",
			"common_searching":    "Searching",
			"common_done":         "Done",
			"common_error":        "Error",
			"common_warning":      "Warning",

			// Config errors
			"config_save_error": "Failed to save config",
			"common_info":       "Info",
			"common_navigation": "Navigation",
			"common_up":         "up",
			"common_down":       "down",
			"common_select":     "select",
			"common_quit":       "quit",
			"common_tip":        "Tip",
			"common_status":     "Status",

			// UI - Install
			"install_title":     "Software Installation",
			"install_progress":  "Installation Progress",
			"install_total":     "Total",
			"install_completed": "Completed",
			"install_failed":    "Failed",
			"install_skipped":   "Skipped",
			"install_select":    "Select software to install",
			"install_confirm":   "Confirm installation of the following software?",

			// UI - Search
			"search_title":       "Search Software",
			"search_placeholder": "Enter software name to search...",
			"search_results":     "Search Results",
			"search_no_results":  "No results found",
			"search_add":         "Add to list",

			"search_refine":      "refine search",
			"search_added_hint":  "Added selected package to configuration",
			"search_added_count": "Added to configuration",

			// UI - Config
			"config_title":    "Configuration",
			"config_add":      "Add Software",
			"config_remove":   "Remove Software",
			"config_edit":     "Edit Software",
			"config_save":     "Save Config",
			"config_load":     "Load Config",
			"config_name":     "Name",
			"config_id":       "ID/Package",
			"config_category": "Category",

			"config_required":          "Name and ID are required",
			"config_added":             "Added",
			"config_updated":           "Updated",
			"config_removed":           "Removed",
			"config_remove_confirm":    "Confirm removal of this package?",
			"config_invalid_selection": "Invalid selection, unable to update",

			// UI - Status
			"status_title":       "System Status",
			"status_platform":    "Platform",
			"status_package_mgr": "Package Manager",
			"status_installed":   "Installed",
			"status_available":   "Available",
			"status_unavailable": "Not available",
			"status_version":     "Version",
			"status_total":       "Total",
			"status_more":        "and",
			"status_packages":    "more packages",
			"status_config":      "Configuration",
			"status_config_path": "Config file path",
			"status_configured":  "Configured packages",
			"status_install_pm":  "Please install a package manager first",

			// UI - Wizard
			"wizard_welcome":           "Welcome to SwiftInstall",
			"wizard_desc":              "This wizard will help you quickly install common software",
			"wizard_step":              "Step",
			"wizard_finish":            "Complete Installation",
			"wizard_select_categories": "Select software categories to install:",
			"wizard_confirm":           "Confirm your selection:",

			// UI - Update
			"update_current":      "Current version",
			"update_latest":       "Latest version",
			"update_checking":     "Checking for updates...",
			"update_failed":       "Failed to check for updates",
			"update_parse_failed": "Failed to parse update info",
			"update_uptodate":     "You are using the latest version!",
			"update_available":    "New version available!",
			"update_download":     "Download URL",
			"update_manual":       "Please check for updates manually",
			"update_hint":         "Download the new version and replace the current executable to update",

			// UI - Clean
			"clean_scanning":  "Scanning cache...",
			"clean_confirm":   "Clean the above cache? [y/N]: ",
			"clean_cleaning":  "Cleaning...",
			"clean_no_cache":  "No cache found to clean",
			"clean_cancelled": "Clean cancelled",
			"clean_partial":   "Some cache cleaning failed:",
			"clean_done":      "Cache cleaned successfully!",

			"about_title":   "About SwiftInstall",
			"about_author":  "Author",
			"about_contact": "Contact",
		},
	}
)

// T 获取翻译文本
func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if lang, ok := translations[currentLang]; ok {
		if text, ok := lang[key]; ok {
			return text
		}
	}

	//  fallback to English
	if lang, ok := translations["en"]; ok {
		if text, ok := lang[key]; ok {
			return text
		}
	}

	return key
}

// SetLanguage 设置当前语言
func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()

	if lang == "zh" || lang == "en" {
		currentLang = lang
	}
}

// GetLanguage 获取当前语言
func GetLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}
