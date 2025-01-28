package main

// 用于匹配应用名，并替换成一些更有意思的输出
func matchApplicationPatterns(wmName, wmClass string) (newAppName string, message string) {
	switch wmClass {
	case "zen":
		return "Zen", "正在使用 Zen 上网冲浪🏄‍♂️"
	case "Code":
		return "VSCode", "正在使用 VSCode 写代码👨‍💻"
	case "CherryStudio":
		return "CherryStudio", "正在与 AI 激情对线🤖"
	case "tabby":
		return "Tabby", "正在使用 Tabby 终端"
	}

	return wmClass, ""
}
