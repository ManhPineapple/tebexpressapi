package utils

import (
	"net/url"
	"regexp"
	"strings"
	"tebexpressapi/pkg/constant"
	"time"
)

var regexpNotAlpha = regexp.MustCompile(`[\s0-9\\\!\@\#\$\%\^|&\*\(\)\_\+\=\_\]\{\}\[\'\:\"\;\/\?\>\.\,\<\/\-]+`)
var regexpNotAlphaNumber = regexp.MustCompile(`^[A-Za-z0-9]*$`)
var regexpNotAlphaNumberSpace = regexp.MustCompile(`[\\\!\@\#\$\%\^|&\*\(\)\_\+\=\_\]\{\}\[\'\:\"\;\/\?\>\.\,\<\/\-]+`)

var regexpNotSpace = regexp.MustCompile(`^[ ].*|[ ]$`)

var regexpDate = regexp.MustCompile("^([0-2][0-9]|(3)[0-1])(\\/)(((0)[0-9])|((1)[0-2]))(\\/)\\d{4}$")
var regexpURL = regexp.MustCompile("^(?:http(s)?:\\/\\/)?[\\w.-]+(?:\\.[\\w\\.-]+)+[\\w\\-\\._~:\\/?#[\\]@%!\\$&'\\(\\)\\*\\+,;=.]+$")
var regexToken = regexp.MustCompile("^[A-Za-z0-9-_=.+/]*$")
var regexpSlug = regexp.MustCompile("^[a-zA-Z0-9-_,\\s]*$")
var regexpEmail = regexp.MustCompile(`^[a-zA-Z0-9_\.]{1,32}@[a-z0-9\-]{2,}(\.[a-z0-9]{2,7}){1,2}$`)
var regexPhoneNumber = regexp.MustCompile("^[+]?[0-9]{1,20}$")
var regexStripTag = regexp.MustCompile("<[^>]*>$")
var regexBarcode = regexp.MustCompile(`^(?:\d+-|)([A|B]\d+(?:-\d+|$))`)
var regexViCharSet = regexp.MustCompile("[^A-Za-z\\d@$!%*#?&_\\/\\-+=() ]")
var regexFileNameCSV = regexp.MustCompile(`^[a-z0-9-]+\.csv$`)
var regexFileNameXLSX = regexp.MustCompile(`^[a-z0-9-]+\.xlsx$`)
var regexpUserName = regexp.MustCompile("^[A-Za-z0-9-_]*$")
var regexpFullName = regexp.MustCompile("^([^0-9!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~]*){0,150}$")

var regexpDomain = regexp.MustCompile(`\b((xn--)?[a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}\b`)
var regexpDesignName = regexp.MustCompile("^[A-Za-z0-9-_]*$")
var regexFloat = regexp.MustCompile(`^\d+\.?\d*$`)

// InvalidNotSpace return true is invalid
func InvalidNotSpace(str string) bool {
	return regexpNotSpace.MatchString(str)
}

// ValidFullName return true is valid
func ValidFullName(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexpFullName.MatchString(trimSpace)
}

// ValidDesignName return true is valid
func ValidDesignName(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexpDesignName.MatchString(trimSpace)
}

// InvalidAlpha return true is invalid
func InvalidAlpha(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexpNotAlpha.MatchString(trimSpace)
}

// ValidAlphaNumberWithoutSpace return true is valid
func ValidAlphaNumberWithoutSpace(str string) bool {
	return regexpNotAlphaNumber.MatchString(str)
}

// InvalidAlphaNumber return true is invalid
func InvalidAlphaNumber(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexpNotAlphaNumberSpace.MatchString(trimSpace)
}

// ValidDate format:31/12/2012
func ValidDate(date string) bool {
	trimSpace := strings.TrimSpace(date)
	return regexpDate.Match([]byte(trimSpace))
}

// ValidURL return is valid
func ValidURL(URL string) bool {
	trimSpace := strings.TrimSpace(URL)
	return regexpURL.Match([]byte(trimSpace))
}

// ValidToken return false is invalid
func ValidToken(token string) bool {
	trimSpace := strings.TrimSpace(token)
	return regexToken.Match([]byte(trimSpace))
}

// ValidFileName csv or xlxs return  true is valid
func ValidFileName(fileName string) bool {
	return regexFileNameCSV.Match([]byte(fileName)) || regexFileNameXLSX.Match([]byte(fileName))
}

// ValidSlug return true is valid
func ValidSlug(slug string) bool {
	trimSpace := strings.TrimSpace(slug)
	return regexpSlug.Match([]byte(trimSpace))
}

// ValidEmail return true is valid
func ValidEmail(email string) bool {
	trimSpace := strings.TrimSpace(email)
	return regexpEmail.Match([]byte(trimSpace))
}

// IsURL return true is url
func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// ValidPhoneNumber return true is valid
func ValidPhoneNumber(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexPhoneNumber.MatchString(trimSpace)
}

func ValidShopName(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexStripTag.MatchString(trimSpace)
}

// ValidUserName return true is invalid
func ValidUserName(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexpUserName.MatchString(trimSpace)
}

// Valid Vietnamese Legacy Character Encodings return true is invalid
func InvalidViCharSet(str string) bool {
	return regexViCharSet.MatchString(str)
}

// valid Domain
func ValidDomain(str *string) bool {
	trimSpace := strings.TrimSpace(*str)
	return regexpDomain.MatchString(trimSpace)
}

// InvalidTag return true is invalid
func InvalidTag(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexStripTag.MatchString(trimSpace)
}

func ParseRawDateTime(rawDateTime string) (t *time.Time) {
	if len(rawDateTime) > 0 {
		for _, l := range []string{
			constant.RequestDateTimeLayout,
			constant.RequestDateTimeLayout2,
			constant.RequestDateTimeLayout3,
			constant.RequestDateTimeLayout4,
			constant.RequestDateTimeLayout5} {

			timeParsed, err := time.Parse(l, rawDateTime)
			if err == nil {
				return &timeParsed
			}
		}
	}

	return nil
}

func ValidBarcode(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexBarcode.MatchString(trimSpace)
}

func ValidatePassword(password string) string {
	if password == "" {
		return "Mật khẩu không được để trống"
	} else if len(password) < 4 {
		return "Mật khẩu tối thiểu 4 ký tự"
	} else if InvalidTag(password) {
		return "Your password not contain special characters."
	}
	return ""
}

func ValidateFloat(str string) bool {
	trimSpace := strings.TrimSpace(str)
	return regexFloat.MatchString(trimSpace)
}
