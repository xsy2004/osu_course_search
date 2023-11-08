package main

import (
	"fmt"
	"strconv"
)

func parseTerm(code int) string {
	// 将数字转换为字符串以便处理
	codeStr := strconv.Itoa(code)

	// 提取年份和学期代码
	yearCode := codeStr[1:3]
	termCode := codeStr[3]

	// 根据末尾的数字确定学期
	var term string
	switch termCode {
	case '2':
		term = "Spring"
	case '4':
		term = "Summer"
	case '8':
		term = "Autumn"
	default:
		fmt.Errorf("无效的学期代码: %v", termCode)
	}

	// 组合完整的年份
	year, err := strconv.Atoi(yearCode)
	if err != nil {
		fmt.Errorf("年份转换错误: %v", err)
	}
	fullYear := 2000 + year

	// 返回格式化的字符串
	return fmt.Sprintf("%s %d", term, fullYear)
}
