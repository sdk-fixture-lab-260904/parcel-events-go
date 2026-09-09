package kernel

import (
	"net/netip"
	"regexp"
	"strconv"
)

var dateFormatPattern = regexp.MustCompile(`^([0-9]{4})-([0-9]{2})-([0-9]{2})$`)
var timeFormatPattern = regexp.MustCompile(`^([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.[0-9]+)?(?:[zZ]|([+-])([0-9]{2}):([0-9]{2}))$`)
var uuidFormatPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func formatNumber(value string) int {
	number, _ := strconv.Atoi(value) // Regex captures contain bounded ASCII digits or an absent offset.
	return number
}
func validDateFormat(value string) bool {
	match := dateFormatPattern.FindStringSubmatch(value)
	if match == nil {
		return false
	}
	year, month, day := formatNumber(match[1]), formatNumber(match[2]), formatNumber(match[3])
	days := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
		days[1] = 29
	}
	return month >= 1 && month <= 12 && day >= 1 && day <= days[month-1]
}
func validTimeFormat(value string) bool {
	match := timeFormatPattern.FindStringSubmatch(value)
	if match == nil {
		return false
	}
	hour, minute, second := formatNumber(match[1]), formatNumber(match[2]), formatNumber(match[3])
	offsetHour, offsetMinute := formatNumber(match[5]), formatNumber(match[6])
	if hour > 23 || minute > 59 || second > 60 || offsetHour > 23 || offsetMinute > 59 {
		return false
	}
	offset := offsetHour*60 + offsetMinute
	if match[4] == "-" {
		offset = -offset
	}
	return second != 60 || (hour*60+minute-offset+1440)%1440 == 1439
}
func matchesFormat(format, value string) bool {
	switch format {
	case "ipv4", "ipv6":
		address, err := netip.ParseAddr(value)
		return err == nil && address.Zone() == "" && ((format == "ipv4" && address.Is4()) || (format == "ipv6" && address.Is6()))
	case "date":
		return validDateFormat(value)
	case "time":
		return validTimeFormat(value)
	case "date-time":
		return len(value) >= 20 && (value[10] == 't' || value[10] == 'T') && validDateFormat(value[:10]) && validTimeFormat(value[11:])
	case "uuid":
		return uuidFormatPattern.MatchString(value)
	default:
		return true
	}
}
